#!/usr/bin/env node
/*
 * sortinggen — 生成外检工位的打标机点表（Macsa）。
 *
 * 数据来源（逐字节核对，不猜）：
 *   - O17006/setup/udt6Macsa.js   6 头 / 4 段
 *   - O17006/setup/udt12Macsa.js  12 头 / 2 段
 *   - O17006/setup/udt16Macsa.js  16 头 / 2 段
 *   - h028.sql sortings.settings.plc.dbs.Sorting（DB 号与偏移）
 *
 * 布局规律（三种头数一致，只是数组长度不同）：
 *
 *   偏移 0            Start[1..N]     INT × N   PLC→PC 请求打印
 *   偏移 2N           Id_Module       DINT      PLC→PC 当前模块号
 *   偏移 2N+4         NumberModule    INT
 *   偏移 2N+6         Step            INT       PLC→PC 当前段序
 *   偏移 2N+8         Enabled[1..N]   INT × N
 *   偏移 4N+8         Alarm[1..N]     INT × N
 *   偏移 6N+8 (+4)    Done[1..N]      INT × N   PC→PLC 打印完成  ← 注意中间有 4 字节对齐空隙
 *   末尾              ModuleRead      INT       PC→PLC 模块数据已读
 *
 * 输出：
 *   configs/devices/instances/sorting_laser.yaml
 *   configs/devices/instances/sorting_laser_golden.json
 *
 * 用法：
 *   node tools/plcgen/sorting.js [--hive <Hive根目录>] [--out <输出目录>]
 */

const fs = require('fs');
const path = require('path');

function parseArgs(argv) {
	const args = { hive: null, out: null };
	for (let i = 2; i < argv.length; i += 1) {
		if (argv[i] === '--hive') args.hive = argv[++i];
		else if (argv[i] === '--out') args.out = argv[++i];
	}
	return args;
}

/**
 * 从 UDT 源码提取字段的真实偏移，作为生成结果的对照基准。
 *
 * 返回值形如 { 12: { start: 0, idModule: 24, step: 30, enabled: 32, alarm: 56, done: 84, moduleRead: 108, size: 110 } }
 */
function extractLayout(udtPath) {
	const src = fs.readFileSync(udtPath, 'utf8');
	const re = /name:\s*'([^']+)',\s*dataBlock:\s*'[^']*',\s*dataType:\s*'([^']+)',\s*byteOffset:\s*'(\d+)'/g;
	const fields = [];
	let m;
	while ((m = re.exec(src)) !== null) {
		fields.push({ name: m[1], type: m[2], offset: Number(m[3]) });
	}

	const find = (suffix) => {
		const f = fields.find((x) => x.name.endsWith(suffix));
		return f ? f.offset : null;
	};

	const sizeMatch = src.match(/size:\s*(\d+)/);
	const layout = {
		start: find('LaserPrinter.Start[1]'),
		idModule: find('LaserPrinter.Id_Module'),
		numberModule: find('LaserPrinter.NumberModule'),
		step: find('LaserPrinter.Step'),
		enabled: find('LaserPrinter.Enabled[1]'),
		alarm: find('LaserPrinter.Alarm[1]'),
		done: find('PC_to_PLC.LaserPrinter.Done[1]'),
		moduleRead: find('LaserPrinter.ModuleRead'),
		size: sizeMatch ? Number(sizeMatch[1]) : null,
	};

	for (const [k, v] of Object.entries(layout)) {
		if (v === null) throw new Error(`${path.basename(udtPath)}: 缺少 ${k}`);
	}
	return { layout, fields };
}

/**
 * 由头数推算出与 UDT 一致的布局，并与 UDT 实际偏移比对。
 *
 * 任何一处不一致都直接失败：宁可生成器报错，也不能产出一份
 * "看起来正常但地址错位"的点表——那会让现场打出错标签。
 */
function deriveLayout(heads) {
	return {
		start: 0,
		idModule: 2 * heads,
		numberModule: 2 * heads + 4,
		step: 2 * heads + 6,
		enabled: 2 * heads + 8,
		alarm: 4 * heads + 8,
		// Done 前有 4 字节对齐空隙（12 头时 80..84），这是 PLC 侧 UDT 的真实布局
		done: 6 * heads + 12,
		// ModuleRead 紧跟在 Done 之后：Done 起点 + N 个 INT
		moduleRead: 8 * heads + 12,
		// UDT 声明大小含尾部填充，只保证"不小于实际字段末端"，不参与严格比对
		size: 8 * heads + 14,
	};
}

function verify(actual, derived, heads) {
	const diffs = [];
	for (const key of Object.keys(derived)) {
		if (key === 'size') {
			// 声明大小允许有尾部对齐填充，只要不小于字段末端即视为一致
			if (actual.size < derived.size) {
				diffs.push(`size: UDT=${actual.size} 小于字段末端 ${derived.size}`);
			}
			continue;
		}
		if (actual[key] !== derived[key]) {
			diffs.push(`${key}: UDT=${actual[key]} 推导=${derived[key]}`);
		}
	}
	if (diffs.length) {
		throw new Error(`${heads} 头布局与 UDT 不一致：\n  ${diffs.join('\n  ')}`);
	}
}

function yamlDevice(code, host, db, heads, stepsPerHead) {
	const L = deriveLayout(heads);
	const arr = (name, start, label) => [
		`      - name: "${name}"`,
		`        db: ${db}`,
		`        start: ${start}`,
		`        length: ${heads * 2}`,
		`        repeat:`,
		`          count: ${heads}`,
		`          stride: 2`,
		`        points:`,
		`          - { name: "${label}", type: "int16", direction: "read" }`,
	];

	const lines = [];
	lines.push(`  - code: "${code}"`);
	lines.push(`    model: "${db === 8012 ? 'macsa_winder' : 'macsa_winder'}"`);
	lines.push(`    host: "${host}"`);
	lines.push(`    port: 102`);
	lines.push(`    rack: 0`);
	lines.push(`    slot: 1`);
	lines.push(`    protocol: "s7"`);
	lines.push(`    description: "外检打标机 ${heads} 头（Macsa，DB${db}）"`);
	lines.push(`    poll_groups:`);
	lines.push(...arr('laser_start', L.start, 'PLC_to_PC.LaserPrinter.Start'));
	lines.push(`      - name: "laser_ctrl"`);
	lines.push(`        db: ${db}`);
	lines.push(`        start: ${L.idModule}`);
	lines.push(`        length: 8`);
	lines.push(`        points:`);
	lines.push(`          - { name: "PLC_to_PC.LaserPrinter.Id_Module", offset: 0, type: "int32", direction: "read" }`);
	lines.push(`          - { name: "PLC_to_PC.LaserPrinter.NumberModule", offset: 4, type: "int16", direction: "read" }`);
	lines.push(`          - { name: "PLC_to_PC.LaserPrinter.Step", offset: 6, type: "int16", direction: "read" }`);
	lines.push(...arr('laser_enabled', L.enabled, 'PLC_to_PC.LaserPrinter.Enabled'));
	lines.push(...arr('laser_alarm', L.alarm, 'PLC_to_PC.LaserPrinter.Alarm'));
	lines.push(...arr('laser_done', L.done, 'PC_to_PLC.LaserPrinter.Done'));
	lines.push(`      - name: "laser_read"`);
	lines.push(`        db: ${db}`);
	lines.push(`        start: ${L.moduleRead}`);
	lines.push(`        length: 2`);
	lines.push(`        points:`);
	lines.push(`          - { name: "PC_to_PLC.LaserPrinter.ModuleRead", type: "int16", direction: "write" }`);

	return { text: lines.join('\n'), layout: L, stepsPerHead };
}

function main() {
	const args = parseArgs(process.argv);
	const hive = path.resolve(args.hive || path.join(__dirname, '..', '..', '..', 'Hive'));
	const out = path.resolve(args.out || path.join(__dirname, '..', '..', 'configs', 'devices', 'instances'));

	const setups = path.join(hive, 'O17006', 'setup');
	const variants = [
		{ heads: 6, stepsPerHead: 4, file: 'udt6Macsa.js', code: 'sorting-laser-6h' },
		{ heads: 12, stepsPerHead: 2, file: 'udt12Macsa.js', code: 'sorting-laser-12h' },
		{ heads: 16, stepsPerHead: 2, file: 'udt16Macsa.js', code: 'sorting-laser-16h' },
	];

	// 旧库 h028 中外检1 的打标机 DB 与 PLC 主机
	const host = '192.168.5.111';
	const db = 8012;

	const header = [
		'# 由 tools/plcgen/sorting.js 从 O17006/setup/udt*Macsa.js 生成，请勿手改。',
		'# 现场修改请通过中心"现场配置"页面：这份文件只作为初始导入数据。',
		'# 主机与 DB 号来自 h028.sql sortings.settings.plc.dbs.Sorting（外检1）。',
		'devices:',
	];

	const golden = { source: 'O17006/setup/udt*Macsa.js + h028.sql sortings', devices: [] };
	const blocks = [];

	for (const v of variants) {
		const udtPath = path.join(setups, v.file);
		if (!fs.existsSync(udtPath)) {
			throw new Error(`找不到 ${udtPath}`);
		}
		const { layout } = extractLayout(udtPath);
		const derived = deriveLayout(v.heads);
		verify(layout, derived, v.heads);

		const dev = yamlDevice(v.code, host, db, v.heads, v.stepsPerHead);
		blocks.push(dev.text);
		golden.devices.push({
			code: v.code,
			heads: v.heads,
			stepsPerHead: v.stepsPerHead,
			db,
			layout: dev.layout,
			udtLayout: layout,
		});
		console.log(`  ${v.code.padEnd(20)} ${v.heads} 头  size=${layout.size}  done@${layout.done}  moduleRead@${layout.moduleRead}`);
	}

	fs.writeFileSync(path.join(out, 'sorting_laser.yaml'), [...header, ...blocks].join('\n') + '\n', 'utf8');
	fs.writeFileSync(path.join(out, 'sorting_laser_golden.json'), JSON.stringify(golden, null, 2), 'utf8');

	console.log(`已生成 ${out}`);
}

main();
