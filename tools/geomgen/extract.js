#!/usr/bin/env node
/*
 * geomgen — 从遗留 Hive 系统提取"物理几何表"，生成冻结的 YAML 配置。
 *
 * 提取来源：
 *   1. loading_bobbins_sequence   ← Hive/DataBase/h028.sql 中的 INSERT
 *   2. getBobbinPosition          ← Hive O17006/setup/udt{6,12,16}Macsa.js
 *
 * 输出：
 *   configs/geometry/loading_sequences.yaml
 *   configs/geometry/head_maps.yaml
 *   configs/geometry/legacy_golden.json   ← 用于 Go 侧逐条比对
 *
 * 用法：
 *   node tools/geomgen/extract.js [--hive <Hive根目录>] [--out <输出目录>]
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

// ---------------------------------------------------------------------------
// 1. loading_bobbins_sequence
// ---------------------------------------------------------------------------

/**
 * 从 h028.sql 提取 loading_bobbins_sequence 的全部行。
 * 返回 [{ tunnelSide, spindlePlace, spindle, newPlace }]
 */
function extractLoadingSequence(sqlPath) {
	const sql = fs.readFileSync(sqlPath, 'utf8');
	const marker = 'INSERT INTO `loading_bobbins_sequence` VALUES';
	const start = sql.indexOf(marker);
	if (start === -1) {
		throw new Error(`找不到 loading_bobbins_sequence 的 INSERT：${sqlPath}`);
	}
	const end = sql.indexOf(';', start);
	if (end === -1) {
		throw new Error('loading_bobbins_sequence 的 INSERT 没有结束分号');
	}
	const body = sql.slice(start + marker.length, end);

	// 形如 ('dx',6,'A',13)
	const tupleRe = /\(\s*'([a-z]+)'\s*,\s*(\d+)\s*,\s*'([AB])'\s*,\s*(\d+)\s*\)/g;
	const rows = [];
	let m;
	while ((m = tupleRe.exec(body)) !== null) {
		rows.push({
			tunnelSide: m[1],
			spindlePlace: Number(m[2]),
			spindle: m[3],
			newPlace: Number(m[4]),
		});
	}
	if (rows.length === 0) {
		throw new Error('loading_bobbins_sequence 未解析到任何行');
	}
	return rows;
}

/**
 * 把行按 tunnel_side 分组，每组内按 spindle_place 排列为 {spindle,A,B} 记录。
 */
function groupLoadingSequence(rows) {
	const bySide = new Map();
	for (const r of rows) {
		if (!bySide.has(r.tunnelSide)) bySide.set(r.tunnelSide, new Map());
		const places = bySide.get(r.tunnelSide);
		if (!places.has(r.spindlePlace)) {
			places.set(r.spindlePlace, { spindlePlace: r.spindlePlace, A: null, B: null });
		}
		places.get(r.spindlePlace)[r.spindle] = r.newPlace;
	}

	const patterns = [];
	for (const [side, places] of [...bySide.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
		const map = [...places.values()].sort((a, b) => a.spindlePlace - b.spindlePlace);
		patterns.push({ code: side, map });
	}
	return patterns;
}

/**
 * 校验一组映射是否为 1..capacity 的完整排列（A、B 两列合并后不重不漏）。
 */
function validatePattern(pattern, capacity) {
	const seen = new Map();
	const problems = [];
	for (const row of pattern.map) {
		for (const side of ['A', 'B']) {
			const v = row[side];
			if (v === null || v === undefined) {
				problems.push(`spindlePlace=${row.spindlePlace} 缺少 ${side} 侧`);
				continue;
			}
			if (v < 1 || v > capacity) {
				problems.push(`spindlePlace=${row.spindlePlace} ${side}=${v} 超出 1..${capacity}`);
			}
			if (seen.has(v)) {
				problems.push(`位置 ${v} 重复（spindlePlace=${seen.get(v)} 与 ${row.spindlePlace}）`);
			}
			seen.set(v, row.spindlePlace);
		}
	}
	for (let v = 1; v <= capacity; v += 1) {
		if (!seen.has(v)) problems.push(`位置 ${v} 缺失`);
	}
	return problems;
}

// ---------------------------------------------------------------------------
// 2. getBobbinPosition
// ---------------------------------------------------------------------------

/**
 * 加载遗留 UDT 模块，穷举 (index, step) 得到映射。
 * 由于 legacy 函数对越界输入没有兜底，我们用"探测直到返回 undefined"的方式确定步数。
 */
function extractHeadMap(udtPath) {
	const mod = require(path.resolve(udtPath));
	if (typeof mod.getBobbinPosition !== 'function') {
		throw new Error(`${udtPath} 没有导出 getBobbinPosition`);
	}
	// prefix/size 用来推断头数：size 与变量个数绑定，这里直接读变量名里的最大下标。
	const variables = mod.variables && mod.variables.variables ? mod.variables.variables : [];
	let headCount = 0;
	for (const v of variables) {
		const m = /Start\[(\d+)\]/.exec(v.name);
		if (m) headCount = Math.max(headCount, Number(m[1]));
	}
	if (headCount === 0) {
		throw new Error(`${udtPath} 未能推断打标头数量`);
	}

	const entries = [];
	for (let head = 1; head <= headCount; head += 1) {
		let step = 1;
		let last = mod.getBobbinPosition(head, step);
		if (last === undefined) {
			throw new Error(`${udtPath} head=${head} step=1 无映射`);
		}
		while (last !== undefined) {
			entries.push({ head, step, position: last });
			step += 1;
			last = mod.getBobbinPosition(head, step);
		}
	}

	const stepsPerHead = Math.max(...entries.map((e) => e.step));
	const positions = entries.map((e) => e.position).sort((a, b) => a - b);
	const capacity = Math.max(...positions);
	return {
		headCount,
		stepsPerHead,
		capacity,
		entries,
		udtSize: mod.variables ? mod.variables.size : null,
	};
}

function validateHeadMap(map) {
	const problems = [];
	const seen = new Set();
	for (const e of map.entries) {
		if (seen.has(e.position)) problems.push(`position ${e.position} 重复`);
		seen.add(e.position);
	}
	for (let p = 1; p <= map.capacity; p += 1) {
		if (!seen.has(p)) problems.push(`position ${p} 缺失`);
	}
	const expected = map.headCount * map.stepsPerHead;
	if (map.entries.length !== expected) {
		problems.push(`映射条目 ${map.entries.length} != 头数×步数 ${expected}`);
	}
	if (map.capacity !== expected) {
		problems.push(`容量 ${map.capacity} != 头数×步数 ${expected}`);
	}
	return problems;
}

// ---------------------------------------------------------------------------
// 3. 输出
// ---------------------------------------------------------------------------

function yamlPatterns(patterns, capacity) {
	const lines = [];
	lines.push('# 由 tools/geomgen/extract.js 从 Hive/DataBase/h028.sql 生成，请勿手改。');
	lines.push('# 语义：卷绕机锭位(spindle_place) + 落筒侧(A/B) → 容器位置(new_place)。');
	lines.push(`capacity: ${capacity}`);
	lines.push('patterns:');
	for (const p of patterns) {
		lines.push(`  - code: "${p.code}"`);
		lines.push('    map:');
		for (const row of p.map) {
			lines.push(`      - { spindle_place: ${row.spindlePlace}, a: ${row.A}, b: ${row.B} }`);
		}
	}
	return lines.join('\n') + '\n';
}

function yamlHeadMaps(maps) {
	const lines = [];
	lines.push('# 由 tools/geomgen/extract.js 从 Hive O17006/setup/udt*Macsa.js 生成，请勿手改。');
	lines.push('# 语义：打标头(head) + 段序(step) → 模块内锭位(position)。');
	lines.push('head_maps:');
	for (const m of maps) {
		lines.push(`  - head_count: ${m.headCount}`);
		lines.push(`    steps_per_head: ${m.stepsPerHead}`);
		lines.push(`    capacity: ${m.capacity}`);
		lines.push('    entries:');
		for (const e of m.entries) {
			lines.push(`      - { head: ${e.head}, step: ${e.step}, position: ${e.position} }`);
		}
	}
	return lines.join('\n') + '\n';
}

function main() {
	const args = parseArgs(process.argv);
	const hive = path.resolve(args.hive || path.join(__dirname, '..', '..', '..', 'Hive'));
	const outDir = path.resolve(args.out || path.join(__dirname, '..', '..', 'configs', 'geometry'));

	const sqlPath = path.join(hive, 'DataBase', 'h028.sql');
	if (!fs.existsSync(sqlPath)) {
		throw new Error(`找不到 ${sqlPath}，请用 --hive 指定 Hive 根目录`);
	}

	// --- loading sequences ---
	const rows = extractLoadingSequence(sqlPath);
	const patterns = groupLoadingSequence(rows);

	// 推导容量：每个 pattern 的 A+B 值集合大小
	const capacities = patterns.map((p) => {
		const vals = new Set();
		for (const row of p.map) {
			if (row.A !== null) vals.add(row.A);
			if (row.B !== null) vals.add(row.B);
		}
		return vals.size;
	});
	const capacity = Math.max(...capacities);

	let failed = false;
	for (const p of patterns) {
		const problems = validatePattern(p, capacity);
		if (problems.length) {
			failed = true;
			console.error(`[FAIL] loading pattern "${p.code}" (${p.map.length} 行):`);
			for (const msg of problems) console.error(`       - ${msg}`);
		}
	}
	if (failed) {
		console.error('存在无效的 loading pattern，已中止输出。');
		process.exit(1);
	}

	// --- head maps ---
	// 遗留部署中 O17006 位于"外检"站点的部署包内，先试常见路径。
	const udtDirs = [
		path.join(hive, 'O17006', 'setup'),
		path.join(hive, '外检', 'HIVE', 'O17006', 'setup'),
		path.join(hive, '..', 'jiatong', '外检', 'HIVE', 'O17006', 'setup'),
	];
	const udtDir = udtDirs.find((d) => fs.existsSync(d));
	if (!udtDir) {
		throw new Error(`找不到 O17006/setup 目录，已尝试：\n  ${udtDirs.join('\n  ')}`);
	}
	const maps = [];
	for (const headCount of [6, 12, 16]) {
		const udtPath = path.join(udtDir, `udt${headCount}Macsa.js`);
		if (!fs.existsSync(udtPath)) {
			console.warn(`[SKIP] 找不到 ${udtPath}`);
			continue;
		}
		const map = extractHeadMap(udtPath);
		if (map.headCount !== headCount) {
			throw new Error(`${udtPath} 推断头数 ${map.headCount} 与文件名不符`);
		}
		const problems = validateHeadMap(map);
		if (problems.length) {
			failed = true;
			console.error(`[FAIL] head map ${headCount} 头:`);
			for (const msg of problems) console.error(`       - ${msg}`);
		}
		maps.push(map);
	}
	if (failed) {
		console.error('存在无效的 head map，已中止输出。');
		process.exit(1);
	}

	fs.mkdirSync(outDir, { recursive: true });
	fs.writeFileSync(path.join(outDir, 'loading_sequences.yaml'), yamlPatterns(patterns, capacity), 'utf8');
	fs.writeFileSync(path.join(outDir, 'head_maps.yaml'), yamlHeadMaps(maps), 'utf8');

	// golden：供 Go 侧逐条比对
	const golden = {
		source: {
			loadingSequence: 'Hive/DataBase/h028.sql',
			headMaps: 'Hive/O17006/setup/udt{6,12,16}Macsa.js',
		},
		loadingCapacity: capacity,
		patterns: patterns.map((p) => ({
			code: p.code,
			map: p.map.map((r) => ({ spindlePlace: r.spindlePlace, a: r.A, b: r.B })),
		})),
		headMaps: maps.map((m) => ({
			headCount: m.headCount,
			stepsPerHead: m.stepsPerHead,
			capacity: m.capacity,
			entries: m.entries,
		})),
	};
	fs.writeFileSync(path.join(outDir, 'legacy_golden.json'), JSON.stringify(golden, null, 2), 'utf8');

	console.log(`已生成 ${outDir}`);
	console.log(`  loading patterns : ${patterns.map((p) => `${p.code}(${p.map.length})`).join(', ')}  容量=${capacity}`);
	console.log(`  head maps        : ${maps.map((m) => `${m.headCount}头×${m.stepsPerHead}步=${m.capacity}`).join(', ')}`);
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
	extractLoadingSequence,
	groupLoadingSequence,
	validatePattern,
	extractHeadMap,
	validateHeadMap,
};
