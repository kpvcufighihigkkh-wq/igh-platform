// siteimport 把仓库里的初始配置导入中心"现场配置"。
//
// 为什么需要这个工具：
//   - 仓field库里的 YAML 只是初始数据，现场配置的唯一权威是中心数据库
//   - 现场人员不该手工把几百行点表敲进页面，需要一个可审计的一次性导入
//
// 导入的内容：
//   - PLC 点表（spinning_sides.yaml / sorting_laser.yaml 等）
//   - 装载位置查表（loading_sequences.yaml）
//   - 外检锭位映射（head_maps.yaml）
//
// 导入前会在中心侧完整校验；校验不通过整份拒绝，不做部分写入。
//
// 用法：
//
//	siteimport -server http://127.0.0.1:8099 -edge sorting-L04 \
//	  -plc configs/devices/instances/sorting_laser.yaml \
//	  -heads configs/geometry/head_maps.yaml
//
//	siteimport ... -dry-run     # 只校验不保存
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/igh-platform/igh-platform/internal/shared/config"
	"github.com/igh-platform/igh-platform/internal/shared/geometry"
	"github.com/igh-platform/igh-platform/internal/shared/siteconfig"
)

type options struct {
	server   string
	edge     string
	user     string
	pass     string
	plcFile  string
	loadFile string
	headFile string
	remark   string
	dryRun   bool
	// replace 为 true 时整份覆盖；默认是合并（只改本次给出的部分）
	replace bool
	timeout time.Duration
}

func main() {
	var opt options
	flag.StringVar(&opt.server, "server", "http://127.0.0.1:8099", "中心服务地址")
	flag.StringVar(&opt.edge, "edge", "", "目标边端编码（必填）")
	flag.StringVar(&opt.user, "user", "admin", "登录用户名")
	flag.StringVar(&opt.pass, "pass", "admin123", "登录密码")
	flag.StringVar(&opt.plcFile, "plc", "", "PLC 点表文件（可多个，用逗号分隔）")
	flag.StringVar(&opt.loadFile, "loading", "", "装载位置查表文件")
	flag.StringVar(&opt.headFile, "heads", "", "外检锭位映射文件")
	flag.StringVar(&opt.remark, "remark", "由 siteimport 导入初始配置", "修改说明")
	flag.BoolVar(&opt.dryRun, "dry-run", false, "只校验不保存")
	flag.BoolVar(&opt.replace, "replace", false,
		"整份覆盖现有配置（默认合并：只覆盖本次给出的部分）")
	flag.DurationVar(&opt.timeout, "timeout", 30*time.Second, "HTTP 超时")
	flag.Parse()

	if opt.edge == "" {
		fmt.Fprintln(os.Stderr, "错误：必须指定 -edge")
		os.Exit(2)
	}
	if opt.plcFile == "" && opt.loadFile == "" && opt.headFile == "" {
		fmt.Fprintln(os.Stderr, "错误：至少指定 -plc / -loading / -heads 之一")
		os.Exit(2)
	}

	doc, err := buildDocument(opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置失败：%v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: opt.timeout}

	token, err := login(client, opt.server, opt.user, opt.pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "登录失败：%v\n", err)
		os.Exit(1)
	}

	// 边端必须已在中心注册。编码写错时如果直接写入，
	// 配置会静默挂到一个不存在的边端下，现场永远等不到下发。
	if err := ensureEdgeExists(client, opt.server, token, opt.edge); err != nil {
		fmt.Fprintf(os.Stderr, "边端检查失败：%v\n", err)
		os.Exit(1)
	}

	// 默认合并：只覆盖本次显式给出的部分。
	//
	// 保存接口是"整份文档替换"，所以只导入 -loading 时若直接提交，
	// 会把边端已有点表整段清空——现场表现为设备全部消失，
	// 而导入命令本身是成功的，极难排查。
	if !opt.replace {
		existing, err := loadExisting(client, opt.server, token, opt.edge)
		if err != nil {
			fmt.Fprintf(os.Stderr, "读取现有配置失败：%v\n", err)
			os.Exit(1)
		}
		if existing != nil {
			mergeDocument(existing, doc, opt)
			doc = existing
			fmt.Printf("合并模式：在此基础上更新 %s\n", changedSections(opt))
		}
	}

	if opt.dryRun {
		problems, err := validate(client, opt.server, token, opt.edge, doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "校验请求失败：%v\n", err)
			os.Exit(1)
		}
		reportProblems(problems)
		return
	}

	version, problems, err := save(client, opt.server, token, opt.edge, doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "保存失败：%v\n", err)
		os.Exit(1)
	}
	if len(problems) > 0 {
		reportProblems(problems)
		os.Exit(1)
	}

	fmt.Printf("导入完成：边端 %s 已生成配置版本 %d\n", opt.edge, version)
	fmt.Println("注意：边端不会自动生效，需现场人员在边端页面点\"应用配置\"。")
}

func reportProblems(problems []string) {
	if len(problems) == 0 {
		fmt.Println("校验通过，可以保存")
		return
	}
	fmt.Fprintf(os.Stderr, "校验未通过（共 %d 项）：\n", len(problems))
	for _, p := range problems {
		fmt.Fprintf(os.Stderr, "  - %s\n", p)
	}
}

// buildDocument 从本地文件构造现场配置文档。
func buildDocument(opt options) (*siteconfig.Document, error) {
	doc := &siteconfig.Document{Remark: opt.remark}

	if opt.plcFile != "" {
		for _, path := range strings.Split(opt.plcFile, ",") {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			instances, err := config.LoadPLCInstances(path)
			if err != nil {
				return nil, err
			}
			doc.PLC = append(doc.PLC, instances.Devices...)
			fmt.Printf("读取点表 %s：%d 台设备\n", path, len(instances.Devices))
		}
	}

	if opt.loadFile != "" {
		ls, err := geometry.LoadLoadingSequences(opt.loadFile)
		if err != nil {
			return nil, err
		}
		doc.Loading = ls.LoadingDocOf()
		fmt.Printf("读取装载查表 %s：%d 套模式，容量 %d\n",
			opt.loadFile, len(doc.Loading.Patterns), doc.Loading.Capacity)
	}

	if opt.headFile != "" {
		hm, err := geometry.LoadHeadMaps(opt.headFile)
		if err != nil {
			return nil, err
		}
		doc.Heads = hm.HeadMapsDocOf()
		fmt.Printf("读取锭位映射 %s：%d 套\n", opt.headFile, len(doc.Heads.HeadMaps))
	}

	return doc, nil
}

// ---------------------------------------------------------------------------
// 合并
// ---------------------------------------------------------------------------

// loadExisting 读取边端当前的现场配置；从未配置过时返回 (nil, nil)。
func loadExisting(c *http.Client, server, token, edge string) (*siteconfig.Document, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/api/v1/edge-nodes/%s/site-config", server, edge), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d：%s", res.StatusCode, string(raw))
	}

	var resp struct {
		Data *siteconfig.Document `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("解析现有配置失败：%w", err)
	}
	return resp.Data, nil
}

// mergeDocument 把本次导入的部分覆盖到现有配置上。
//
// 只覆盖调用方显式给出的部分：带 -plc 才换点表，带 -loading 才换装载表。
// 没给出的部分原样保留，避免"只导装载表导致点表被清空"。
func mergeDocument(base, patch *siteconfig.Document, opt options) {
	if opt.plcFile != "" {
		base.PLC = patch.PLC
	}
	if opt.loadFile != "" {
		base.Loading = patch.Loading
	}
	if opt.headFile != "" {
		base.Heads = patch.Heads
	}
	base.Remark = patch.Remark
	// Version 由服务端递增，这里保持原值即可（会被覆盖）
}

// changedSections 返回本次会改动哪些部分，用于打印。
func changedSections(opt options) string {
	var parts []string
	if opt.plcFile != "" {
		parts = append(parts, "PLC 点表")
	}
	if opt.loadFile != "" {
		parts = append(parts, "装载位置查表")
	}
	if opt.headFile != "" {
		parts = append(parts, "外检锭位映射")
	}
	return strings.Join(parts, "、")
}

// ---------------------------------------------------------------------------
// HTTP
// ---------------------------------------------------------------------------

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func login(c *http.Client, server, user, pass string) (string, error) {
	body, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	resp, err := post(c, server+"/api/v1/auth/login", body, "")
	if err != nil {
		return "", err
	}

	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", fmt.Errorf("解析登录响应失败：%w", err)
	}
	if data.Token == "" {
		return "", fmt.Errorf("登录响应中没有 token：%s", resp.Message)
	}
	return data.Token, nil
}

// ensureEdgeExists 确认边端已在中心注册。
func ensureEdgeExists(c *http.Client, server, token, edge string) error {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/api/v1/edge-nodes/%s", server, edge), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("边端 %q 未注册。配置若挂到不存在的边端，现场永远收不到下发；"+
			"请先在边端启动一次让它自动注册，或用正确的 -edge 编码", edge)
	default:
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("HTTP %d：%s", res.StatusCode, string(raw))
	}
}

func validate(c *http.Client, server, token, edge string, doc *siteconfig.Document) ([]string, error) {
	body, _ := json.Marshal(doc)
	resp, err := post(c, fmt.Sprintf("%s/api/v1/edge-nodes/%s/site-config/validate", server, edge), body, token)
	if err != nil {
		return nil, err
	}
	var data struct {
		OK       bool     `json:"ok"`
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("解析校验响应失败：%w", err)
	}
	return data.Problems, nil
}

func save(c *http.Client, server, token, edge string, doc *siteconfig.Document) (int, []string, error) {
	body, _ := json.Marshal(doc)
	req, err := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/edge-nodes/%s/site-config", server, edge),
		bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	var resp apiResponse
	json.Unmarshal(raw, &resp)

	// 422 是"校验未通过"，属于业务结果，把问题逐条返回
	if res.StatusCode == http.StatusUnprocessableEntity {
		var data struct {
			Problems []string `json:"problems"`
		}
		json.Unmarshal(resp.Data, &data)
		return 0, data.Problems, nil
	}
	if res.StatusCode != http.StatusOK {
		return 0, nil, fmt.Errorf("HTTP %d：%s", res.StatusCode, resp.Message)
	}

	var data struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return 0, nil, fmt.Errorf("解析保存响应失败：%w", err)
	}
	return data.Version, nil, nil
}

func post(c *http.Client, url string, body []byte, token string) (*apiResponse, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	var resp apiResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败（HTTP %d）：%s", res.StatusCode, string(raw))
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d：%s", res.StatusCode, resp.Message)
	}
	return &resp, nil
}
