#!/usr/bin/env node
/*
 * plcgen — 从遗留 Hive 系统生成"真实 PLC 点表"。
 *
 * 数据来源：
 *   - Hive/DataBase/h028.sql 的 spinning_sides 表（主机/DB 号/偏移/容量）
 *   - 落筒 O17004/src/udtTMT.js、udtBarmag.js（字段名/类型/字节偏移）
 *
 * 输出：
 *   configs/devices/instances/spinning_sides.yaml
 *   configs/devices/instances/plc_golden.json   ← 供 Go 契约测试比对
 *   configs/devices/instances/SKIPPED.md         ← 被拒绝生成的线路及原因
 *
 * 设计原则：
 *   1. 不猜。配置与 UDT 出现歧义时，该条线路**不生成**，写入 SKIPPED.md。
 *   2. 校验双胞胎两侧收敛：
 *        offsetEven + udtSize + offsetEndEven == offsetOdd + udtSize + offsetEndOdd
 *   3. 校验每个字段的 [offset, offset+width) 落在 UDT 声明大小内。
 *
 * 用法：
 *   node tools/plcgen/extract.js [--hive <Hive根目录>] [--out <输出目录>]
 */

const fs = require('fs');
const path = require('path');

// ---------------------------------------------------------------------------
// 参数
// ---------------------------------------------------------------------------

function parseArgs(argv) {
	const args = { hive: null, out: null };
	for (let i = 2; i < argv.length; i += 1) {
		if (argv[i] === '--hive') args.hive = argv[++i];
		else if (argv[i] === '--out') args.out = argv[++i];
	}
	return args;
}

// ---------------------------------------------------------------------------
// SQL 解析：提取 spinning_sides 的全部元组
// ---------------------------------------------------------------------------

/**
 * 从 INSERT INTO `spinning_sides` VALUES ...; 中解析出顶层元组数组。
 * 正确处理字符串内的括号、逗号与转义单引号。
 */
function parseTuples(sql) {
	const marker = 'INSERT INTO `spinning_sides` VALUES';
	const start = sql.indexOf(marker);
	if (start === -1) throw new Error('找不到 spinning_sides 的 INSERT');
	const bodyStart = start + marker.length;

	const tuples = [];
	let i = bodyStart;
	while (i < sql.length) {
		// 跳到下一个 '('
		while (i < sql.length && sql[i] !== '(' && sql[i] !== ';') i += 1;
		if (i >= sql.length || sql[i] === ';') break;

		const begin = i;
		let depth = 0;
		let inString = false;
		for (; i < sql.length; i += 1) {
			const ch = sql[i];
			if (inString) {
				if (ch === '\\') { i += 1; continue; } // 跳过转义字符
				if (ch === "'") inString = false;
				continue;
			}
			if (ch === "'") { inString = true; continue; }
			if (ch === '(') { depth += 1; continue; }
			if (ch === ')') {
				depth -= 1;
				if (depth === 0) { i += 1; break; }
			}
		}
		tuples.push(sql.slice(begin + 1, i - 1));
	}
	return tuples;
}

/** 把一个元组的顶层字段切分开（逗号分隔，忽略字符串内与括号内的逗号）。 */
function splitFields(tupleBody) {
	const fields = [];
	let cur = '';
	let inString = false;
	let depth = 0;
	for (let i = 0; i < tupleBody.length; i += 1) {
		const ch = tupleBody[i];
		if (inString) {
			cur += ch;
			if (ch === '\\') { cur += tupleBody[i + 1] || ''; i += 1; continue; }
			if (ch === "'") inString = false;
			continue;
		}
		if (ch === "'") { inString = true; cur += ch; continue; }
		if (ch === '(') { depth += 1; cur += ch; continue; }
		if (ch === ')') { depth -= 1; cur += ch; continue; }
		if (ch === ',' && depth === 0) { fields.push(cur.trim()); cur = ''; continue; }
		cur += ch;
	}
	if (cur.trim() !== '') fields.push(cur.trim());
	return fields;
}

/** 把 SQL 字面量转成 JS 值。 */
function unquote(token) {
	const t = token.trim();
	if (t === 'NULL') return null;
	if (/^'.*'$/s.test(t)) {
		const inner = t.slice(1, -1);
		let out = '';
		for (let i = 0; i < inner.length; i += 1) {
			const ch = inner[i];
			if (ch === '\\') {
				const next = inner[i + 1];
				i += 1;
				if (next === 'n') out += '\n';
				else if (next === 't') out += '\t';
				else if (next === 'r') out += '\r';
				else if (next === '0') out += '\0';
				else out += next; // \" \' \\ 等
				continue;
			}
			out += ch;
		}
		return out;
	}
	const n = Number(t);
	return Number.isNaN(n) ? t : n;
}

// ---------------------------------------------------------------------------
// UDT 加载
// ---------------------------------------------------------------------------

const TYPE_WIDTH = {
	C: null, // 由 arrayLength 决定
	B: 1,
	BYTE: 1,
	INT: 2,
	WORD: 2,
	DINT: 4,
	DWORD: 4,
	REAL: 4,
	TIME: 4,
	X: 1,
};

/** 读取遗留 UDT 模块（CommonJS），返回 { size, fields[] }。 */
function loadUdt(udtPath) {
	const mod = require(path.resolve(udtPath));
	const vars = mod.variables || [];
	const fields = vars.map((v) => {
		const width = TYPE_WIDTH[v.dataType] == null
			? parseInt(v.arrayLength, 10)
			: TYPE_WIDTH[v.dataType];
		return {
			name: v.name,
			type: v.dataType,
			offset: parseInt(v.byteOffset, 10),
			length: width,
			arrayLength: v.arrayLength ? parseInt(v.arrayLength, 10) : null,
			bitOffset: v.bitOffset || '',
		};
	});
	const required = fields.reduce((m, f) => Math.max(m, f.offset + (f.length || 0)), 0);
	return { size: mod.size, fields, required };
}

function validateUdt(udt) {
	const problems = [];
	if (!Number.isFinite(udt.size) || udt.size <= 0) {
		problems.push(`UDT 声明大小无效: ${udt.size}`);
	}
	for (const f of udt.fields) {
		if (!Number.isFinite(f.offset)) {
			problems.push(`字段 ${f.name} 缺少有效 byteOffset`);
			continue;
		}
		if (!Number.isFinite(f.length) || f.length <= 0) {
			problems.push(`字段 ${f.name} 无法确定宽度（type=${f.type}, arrayLength=${f.arrayLength}）`);
			continue;
		}
		if (udt.size && f.offset + f.length > udt.size) {
			problems.push(
				`字段 ${f.name} 范围 [${f.offset}, ${f.offset + f.length}) 超出声明大小 ${udt.size}`
			);
		}
	}
	// 使用 required 而非 declared size 做比较，便于暴露声明偏小的问题
	if (udt.required > udt.size) {
		problems.push(`声明大小 ${udt.size} 小于字段所需 ${udt.required}`);
	}
	return problems;
}

// ---------------------------------------------------------------------------
// 双胞胎偏移解析
// ---------------------------------------------------------------------------

/**
 * 解析 Winder 配置，返回规范化的布局。
 * 支持两种写法：
 *    A. { number, count, offsetEven, offsetEndEven, offsetOdd, offsetEndOdd }
 *    B. { number, count, offset, offsetEnd }（仅单侧）
 */
function normalizeWinder(winder, udtSize) {
	if (!winder) return { error: '缺少 plc.dbs.Winder 配置' };
	const count = Number(winder.count);
	if (!Number.isFinite(count) || count <= 0) {
		return { error: `Winder.count 无效: ${winder.count}` };
	}

	const hasEvenOdd =
		Number.isFinite(Number(winder.offsetEven)) &&
		Number.isFinite(Number(winder.offsetEndEven)) &&
		Number.isFinite(Number(winder.offsetOdd)) &&
		Number.isFinite(Number(winder.offsetEndOdd));

	const hasPlain =
		Number.isFinite(Number(winder.offset)) &&
		Number.isFinite(Number(winder.offsetEnd));

	if (hasEvenOdd) {
		const offsetEven = Number(winder.offsetEven);
		const offsetEndEven = Number(winder.offsetEndEven);
		const offsetOdd = Number(winder.offsetOdd);
		const offsetEndOdd = Number(winder.offsetEndOdd);

		const evenEnd = offsetEven + udtSize + offsetEndEven;
		const oddEnd = offsetOdd + udtSize + offsetEndOdd;
		if (evenEnd !== oddEnd) {
			return {
				error:
					`双胞胎两侧不收敛：Even 终点 ${evenEnd} (${offsetEven}+${udtSize}+${offsetEndEven}) ` +
					`!= Odd 终点 ${oddEnd} (${offsetOdd}+${udtSize}+${offsetEndOdd})`,
			};
		}

		// offsetEven == offsetOdd 说明 Even/Odd 是同一个地址的两个别名，
		// 这块 PLC 是**单面锭位**（如 120 锭线），不是双胞胎。
		// 若仍按左右两侧展开，同一段字节会被读两遍、解释两遍，
		// 每个落筒都会重复上报一次。
		if (offsetEven === offsetOdd && offsetEndEven === offsetEndOdd) {
			return {
				schema: 'even-odd-single',
				count,
				udtSize,
				stride: evenEnd,
				offsetEven,
				offsetEndEven,
				offsetOdd,
				offsetEndOdd,
				sides: null,
			};
		}

		return {
			schema: 'even-odd',
			count,
			udtSize,
			stride: evenEnd,
			offsetEven,
			offsetEndEven,
			offsetOdd,
			offsetEndOdd,
			sides: { left: offsetEven, right: offsetOdd },
		};
	}

	if (hasPlain) {
		const offset = Number(winder.offset);
		const offsetEnd = Number(winder.offsetEnd);
		return {
			schema: 'plain',
			count,
			udtSize,
			stride: offset + udtSize + offsetEnd,
			offsetEven: offset,
			offsetEndEven: offsetEnd,
			offsetOdd: offset,
			offsetEndOdd: offsetEnd,
			sides: null,
		};
	}

	return {
		error:
			`Winder 配置既没有 offsetEven/offsetOdd/offsetEndEven/offsetEndOdd，也没有 offset/offsetEnd。` +
			`原始键: ${Object.keys(winder).join(', ')}`,
	};
}

// ---------------------------------------------------------------------------
// YAML 输出
// ---------------------------------------------------------------------------

function yamlString(s) {
	return `"${String(s).replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;
}

function renderDevice(entry) {
	const L = [];
	L.push(`  - code: ${yamlString(entry.code)}`);
	L.push(`    model: ${yamlString(entry.model)}`);
	L.push(`    host: ${yamlString(entry.host)}`);
	L.push(`    port: ${entry.port}`);
	L.push(`    rack: ${entry.rack}`);
	L.push(`    slot: ${entry.slot}`);
	L.push(`    protocol: "s7"`);
	L.push(`    description: ${yamlString(entry.description)}`);
	L.push(`    twin:`);
	L.push(`      udt_size: ${entry.winder.udtSize}`);
	L.push(`      stride: ${entry.winder.stride}`);
	L.push(`      count: ${entry.winder.count}`);
	L.push(`      offset_even: ${entry.winder.offsetEven}`);
	L.push(`      offset_end_even: ${entry.winder.offsetEndEven}`);
	L.push(`      offset_odd: ${entry.winder.offsetOdd}`);
	L.push(`      offset_end_odd: ${entry.winder.offsetEndOdd}`);
	const sides = entry.winder.sides;
	if (sides) {
		L.push(`      sides:`);
		L.push(`        left:  { offset: ${sides.left} }`);
		L.push(`        right: { offset: ${sides.right} }`);
	} else {
		// 单面锭位：没有左右两侧，每个 stride 就是一个独立机位
		L.push(`      sides: null`);
	}
	L.push(`      semantics: ${yamlString(entry.winder.semantics)}`);
	L.push(`    poll_groups:`);
	L.push(`      - name: "winder_data"`);
	L.push(`        interval_ms: 500`);
	L.push(`        db: ${entry.winderDb}`);
	L.push(`        start: 0`);
	L.push(`        length: ${entry.winder.stride * entry.winder.count}`);
	L.push(`        repeat:`);
	L.push(`          count: ${entry.winder.count}`);
	L.push(`          stride: ${entry.winder.stride}`);
	if (sides) {
		L.push(`          sides:`);
		L.push(`            left:  { offset: ${sides.left} }`);
		L.push(`            right: { offset: ${sides.right} }`);
	}
	L.push(`        points:`);
	for (const f of entry.udt.fields) {
		const bits = [
			`name: ${yamlString(f.name)}`,
			`offset: ${f.offset}`,
			`type: ${yamlString(mapType(f.type))}`,
		];
		if (f.type === 'C') bits.push(`length: ${f.length}`);
		if (f.bitOffset !== '') bits.push(`bit: ${f.bitOffset}`);
		bits.push(`direction: "read"`);
		L.push(`          - { ${bits.join(', ')} }`);
	}
	return L.join('\n');
}

function mapType(legacy) {
	switch (legacy) {
		case 'C': return 'string';
		case 'B': return 'byte';
		case 'BYTE': return 'uint8';
		case 'INT': return 'int16';
		case 'WORD': return 'uint16';
		case 'DINT': return 'int32';
		case 'DWORD': return 'uint32';
		case 'REAL': return 'float32';
		case 'TIME': return 'uint32';
		case 'X': return 'bool';
		default: return legacy.toLowerCase();
	}
}

// ---------------------------------------------------------------------------
// 主流程
// ---------------------------------------------------------------------------

function main() {
	const args = parseArgs(process.argv);
	const hive = path.resolve(args.hive || path.join(__dirname, '..', '..', '..', 'Hive'));
	const outDir = path.resolve(args.out || path.join(__dirname, '..', '..', 'configs', 'devices', 'instances'));

	const sqlPath = path.join(hive, 'DataBase', 'h028.sql');
	if (!fs.existsSync(sqlPath)) {
		throw new Error(`找不到 ${sqlPath}，请用 --hive 指定 Hive 根目录`);
	}
	const sql = fs.readFileSync(sqlPath, 'utf8');

	// 定位落筒 UDT（优先仓库内的 legacy/ 副本，保证可复现）
	const udtDirs = [
		path.join(__dirname, '..', '..', 'legacy', 'O17004', 'src'),
		path.join(hive, 'O17004', 'src'),
		path.join(hive, '落筒', 'HIVE', 'O17004', 'src'),
		path.join(hive, '..', 'jiatong', '落筒', 'HIVE', 'O17004', 'src'),
	];
	const udtDir = udtDirs.find((d) => fs.existsSync(path.join(d, 'udtTMT.js')));
	if (!udtDir) throw new Error(`找不到 O17004/src/udtTMT.js，已尝试：\n  ${udtDirs.join('\n  ')}`);

	const udts = {
		TMT: loadUdt(path.join(udtDir, 'udtTMT.js')),
		BARMAG: loadUdt(path.join(udtDir, 'udtBarmag.js')),
	};

	const udtProblems = {};
	for (const [name, udt] of Object.entries(udts)) {
		udtProblems[name] = validateUdt(udt);
	}

	const tuples = parseTuples(sql);
	const devices = [];
	const golden = { source: 'Hive/DataBase/h028.sql', devices: [] };
	const skipped = [];

	for (const body of tuples) {
		const f = splitFields(body);
		const id = unquote(f[0]);
		let settingsRaw = unquote(f[4]);
		let settings;
		try {
			settings = JSON.parse(settingsRaw);
		} catch (e) {
			skipped.push({ id, name: unquote(f[3]), reason: `settings 不是合法 JSON: ${e.message}` });
			continue;
		}

		const lineName = unquote(f[3]);
		const loadingSide = unquote(f[6]);
		const whType = settings.warehouseType;
		const plc = settings.plc || {};
		const dbs = plc.dbs || {};
		const winder = dbs.Winder;

		const udt = udts[whType];
		if (!udt) {
			skipped.push({ id, name: lineName, reason: `未知 warehouseType: ${whType}` });
			continue;
		}
		if (udtProblems[whType] && udtProblems[whType].length) {
			skipped.push({
				id, name: lineName,
				reason: `UDT ${whType} 自身不一致：${udtProblems[whType].join('；')}`,
			});
			continue;
		}

		const layout = normalizeWinder(winder, udt.size);
		if (layout.error) {
			skipped.push({ id, name: lineName, reason: layout.error });
			continue;
		}

		// 设备编码：从线路名中提取数字，规范化为 winder-L01 / winder-L09
		const lineDigits = String(lineName).match(/(\d+)/);
		const lineCode = lineDigits ? `L${lineDigits[1].padStart(2, '0')}` : `S${id}`;

		devices.push({
			id,
			code: `winder-${lineCode}`,
			model: whType === 'BARMAG' ? 'barmag_winder' : 'tmt_winder',
			description: `${lineName} / ${loadingSide} / ${whType}`,
			host: plc.host,
			port: plc.port || 102,
			rack: plc.rack || 0,
			slot: plc.slot || 1,
			winderDb: Number(winder.number),
			udt,
			winder: {
				...layout,
				semantics: layout.sides
					? '来自 h028.sql spinning_sides.settings.plc.dbs.Winder；' +
					  '两侧收敛于同一 stride，左=Even、右=Odd 为约定，需现场核对标签左右。'
					: '来自 h028.sql spinning_sides.settings.plc.dbs.Winder；' +
					  'offsetEven 与 offsetOdd 相同，判定为单面锭位（无左右之分），' +
					  '每个 stride 是一个独立机位。',
			},
		});
		golden.devices.push({
			id,
			code: devices[devices.length - 1].code,
			host: plc.host,
			winderDb: Number(winder.number),
			winder: layout,
			fields: udt.fields.map((x) => ({ name: x.name, offset: x.offset, length: x.length })),
			udtSize: udt.size,
		});
	}

	if (!devices.length) {
		throw new Error('没有任何线路通过校验，拒绝输出');
	}

	fs.mkdirSync(outDir, { recursive: true });

	const header = [
		'# 由 tools/plcgen/extract.js 从 Hive/DataBase/h028.sql 与 O17004 UDT 生成，请勿手改。',
		'# 生成器遇到歧义会拒绝输出该线路，原因见 SKIPPED.md。',
		'devices:',
	].join('\n');
	const yaml = header + '\n' + devices.map(renderDevice).join('\n') + '\n';
	fs.writeFileSync(path.join(outDir, 'spinning_sides.yaml'), yaml, 'utf8');
	fs.writeFileSync(path.join(outDir, 'plc_golden.json'), JSON.stringify(golden, null, 2), 'utf8');

	const skippedMd = [
		'# 未生成的线路',
		'',
		`生成时间：${new Date().toISOString()}`,
		'',
		'生成器遵循"歧义即拒绝"原则。以下线路未被写入 spinning_sides.yaml：',
		'',
		'| id | 线路 | 原因 |',
		'|---|---|---|',
		...skipped.map((s) => `| ${s.id} | ${s.name} | ${s.reason} |`),
		'',
		'## UDT 自检结果',
		'',
		...Object.entries(udtProblems).map(([n, p]) =>
			p.length ? `- **${n}**：${p.join('；')}` : `- ${n}：通过`),
		'',
	].join('\n');
	fs.writeFileSync(path.join(outDir, 'SKIPPED.md'), skippedMd, 'utf8');

	console.log(`已生成 ${outDir}`);
	console.log(`  devices : ${devices.length}`);
	for (const d of devices) {
		console.log(`    ${d.id} ${d.code}  ${d.host}  DB${d.winderDb}  stride=${d.winder.stride}  count=${d.winder.count}`);
	}
	if (skipped.length) {
		console.log(`  skipped : ${skipped.length}（详见 SKIPPED.md）`);
		for (const s of skipped) console.log(`    ${s.id} ${s.name}: ${s.reason}`);
	}
}

if (require.main === module) {
	try {
		main();
	} catch (err) {
		console.error(`[ERROR] ${err.message}`);
		process.exit(1);
	}
}

module.exports = {
	parseTuples,
	splitFields,
	unquote,
	loadUdt,
	validateUdt,
	normalizeWinder,
};
