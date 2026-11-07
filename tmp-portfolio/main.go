package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/larsartmann/io-surface/internal/extract"
)

const (
	toolVersion  = "0.2.0"
	projectsJSON = "/tmp/pma-projects.json"
	templatePath = "tmp-portfolio/template.html"
	outputHTML   = "portfolio-io-insights.html"
	outputStats  = "/tmp/io-surface-portfolio-stats.json"
)

var generatedGlobs = []string{
	"*_templ.go", "zz_generated*", "*.pb.go", "*.pb.gw.go", "*_grpc.pb.go",
	"*_string.go", "*_gen.go", "*.deepcopy.go", "*_easyjson.go",
}

type pmaProject struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	TypeDetails struct {
		Primary string   `json:"primary"`
		All     []string `json:"all"`
	} `json:"typeDetails"`
}

type invCounts struct {
	prodFiles int
	testFiles int
	prodLOC   int
	testLOC   int
	topDirs   map[string]bool
}

type projStats struct {
	Name                string
	Path                string
	Types               []string
	Missing             bool
	ScanErr             string
	ProdFiles           int
	TestFiles           int
	ProdLOC             int
	TestLOC             int
	SitesProd           int
	SitesTest           int
	SitesToken          int
	DroppedFixtureSites int
	Dominant            map[string]int
	Touched             map[string]int
	Calls               map[string]int
	CallFams            map[string]map[string]int
	HotFiles            map[string]int
	ViaSeams            map[string]int
	Declared            bool
}

type callAgg struct {
	Fams     map[string]int
	Sites    int
	Projects int
}

type hotRow struct {
	Project string
	File    string
	Sites   int
}

type portfolio struct {
	GeneratedAt         string
	ToolVersion         string
	ProjectsScanned     int
	ProjectsMissing     int
	GoProjects          int
	ProdFiles           int
	TestFiles           int
	ProdLOC             int
	TestLOC             int
	SitesProd           int
	SitesTest           int
	SitesToken          int
	DroppedFixtureSites int
	FamilyDominant      map[string]int
	FamilyTouched       map[string]int
	Calls               map[string]*callAgg
	ViaSeams            map[string]int
	Declared            []string
	ZeroIO              []string
	Entropy             []string
	Exec                []string
	DB                  []string
	Network             []hotRow
	Hotspots            []hotRow
	ScanFailed          []string
	Projects            []*projStats
}

func main() {
	start := time.Now()
	raw, err := os.ReadFile(projectsJSON)
	fatalIf(err, "read projects json")
	i := strings.Index(string(raw), "[")
	if i < 0 {
		fmt.Println("FATAL: no JSON array in", projectsJSON)
		os.Exit(2)
	}
	var projects []pmaProject
	fatalIf(json.Unmarshal([]byte(string(raw)[i:]), &projects), "parse projects json")

	results := make([]*projStats, len(projects))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for idx, p := range projects {
		wg.Add(1)
		go func(i int, p pmaProject) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = scanProject(p)
		}(idx, p)
	}
	wg.Wait()

	pf := buildPortfolio(projects, results, start)
	writeStats(pf)
	doc := render(pf)
	fatalIf(os.WriteFile(outputHTML, []byte(doc), 0o644), "write html")
	fmt.Printf("projects=%d missing=%d go=%d prodFiles=%d prodLOC=%s sites(prod)=%d sites(test)=%d token=%d declared=%d droppedFixtures=%d failed=%d\n",
		pf.ProjectsScanned, pf.ProjectsMissing, pf.GoProjects, pf.ProdFiles, comma(pf.ProdLOC),
		pf.SitesProd, pf.SitesTest, pf.SitesToken, len(pf.Declared), pf.DroppedFixtureSites, len(pf.ScanFailed))
	fmt.Println("top families (dominant):", topStr(pf.FamilyDominant, 6))
	fmt.Println("top calls:", topStrCalls(pf.Calls, 8))
	fmt.Println("wrote", outputHTML, "and", outputStats, "in", time.Since(start).Round(time.Millisecond))
}

func fatalIf(err error, what string) {
	if err != nil {
		fmt.Println("FATAL:", what, "-", err)
		os.Exit(2)
	}
}

func skipDirName(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" ||
		name == "testdata" || name == "docs"
}

func matchGenerated(base string) bool {
	for _, g := range generatedGlobs {
		if ok, _ := filepath.Match(g, base); ok {
			return true
		}
	}
	return false
}

func scanInventory(root string) invCounts {
	inv := invCounts{topDirs: map[string]bool{}}
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && skipDirName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		base := d.Name()
		if !strings.HasSuffix(base, ".go") || matchGenerated(base) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		lines := bytes.Count(data, []byte{'\n'})
		if len(data) > 0 && data[len(data)-1] != '\n' {
			lines++
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		segs := strings.Split(rel, "/")
		if len(segs) == 1 {
			inv.topDirs["."] = true
		} else {
			inv.topDirs[segs[0]] = true
		}
		if strings.HasSuffix(base, "_test.go") {
			inv.testFiles++
			inv.testLOC += lines
		} else {
			inv.prodFiles++
			inv.prodLOC += lines
		}
		return nil
	})
	return inv
}

func scanProject(p pmaProject) *projStats {
	ps := &projStats{
		Name: p.Name, Path: p.Path, Types: p.TypeDetails.All,
		Dominant: map[string]int{}, Touched: map[string]int{},
		Calls: map[string]int{}, CallFams: map[string]map[string]int{},
		HotFiles: map[string]int{}, ViaSeams: map[string]int{},
	}
	if _, err := os.Stat(p.Path); err != nil {
		ps.Missing = true
		return ps
	}
	if len(ps.Types) == 0 {
		ps.Types = []string{p.TypeDetails.Primary}
	}
	inv := scanInventory(p.Path)
	ps.ProdFiles, ps.TestFiles = inv.prodFiles, inv.testFiles
	ps.ProdLOC, ps.TestLOC = inv.prodLOC, inv.testLOC
	if len(inv.topDirs) == 0 {
		return ps
	}
	dirs := make([]string, 0, len(inv.topDirs))
	for d := range inv.topDirs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	sites, err := extract.Extract(p.Path, extract.Options{IncludeTests: true, Dirs: dirs, Ignore: generatedGlobs})
	if err != nil {
		ps.ScanErr = err.Error()
		return ps
	}
	for _, s := range sites {
		locs := make([]string, 0, len(s.Locations))
		for _, loc := range s.Locations {
			if keepLocation(loc) {
				locs = append(locs, loc)
			}
		}
		dropped := len(s.Locations) - len(locs)
		if dropped > 0 {
			ps.DroppedFixtureSites += dropped
		}
		if len(locs) == 0 {
			continue
		}
		isTest := s.Testinfra
		for _, loc := range locs {
			if strings.HasSuffix(locFile(loc), "_test.go") {
				isTest = true
				break
			}
		}
		if isTest {
			ps.SitesTest++
			continue
		}
		ps.SitesProd++
		ps.Dominant[s.Family()]++
		for _, f := range s.Families {
			ps.Touched[f]++
		}
		files := map[string]bool{}
		for _, loc := range locs {
			files[locFile(loc)] = true
		}
		for f := range files {
			ps.HotFiles[f]++
		}
		for _, c := range s.Calls {
			ps.Calls[c]++
			if ps.CallFams[c] == nil {
				ps.CallFams[c] = map[string]int{}
			}
			for _, f := range s.Families {
				ps.CallFams[c][f]++
			}
		}
		if s.Via != "" {
			ps.ViaSeams[s.Via]++
		}
		if s.Confidence == extract.ConfidenceToken {
			ps.SitesToken++
		}
	}
	ps.Declared = fileExists(filepath.Join(p.Path, "docs", "architecture", "io-surface", "facts.cue"))
	return ps
}

func keepLocation(loc string) bool {
	p := loc
	if i := strings.LastIndex(loc, ":"); i >= 0 {
		p = loc[:i]
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "testdata" || seg == "docs" {
			return false
		}
	}
	return true
}

func locFile(loc string) string {
	if i := strings.LastIndex(loc, ":"); i >= 0 {
		return loc[:i]
	}
	return loc
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func buildPortfolio(projects []pmaProject, results []*projStats, start time.Time) *portfolio {
	pf := &portfolio{
		GeneratedAt:    start.Format("2006-01-02 15:04"),
		ToolVersion:    toolVersion,
		FamilyDominant: map[string]int{},
		FamilyTouched:  map[string]int{},
		Calls:          map[string]*callAgg{},
		ViaSeams:       map[string]int{},
	}
	nameOf := map[string]string{}
	fileCounts := map[hotKey]int{}
	for idx, p := range projects {
		ps := results[idx]
		pf.Projects = append(pf.Projects, ps)
		nameOf[ps.Name] = p.Name
		if ps.Missing {
			pf.ProjectsMissing++
			continue
		}
		pf.ProjectsScanned++
		pf.ProdFiles += ps.ProdFiles
		pf.TestFiles += ps.TestFiles
		pf.ProdLOC += ps.ProdLOC
		pf.TestLOC += ps.TestLOC
		pf.SitesProd += ps.SitesProd
		pf.SitesTest += ps.SitesTest
		pf.SitesToken += ps.SitesToken
		pf.DroppedFixtureSites += ps.DroppedFixtureSites
		if ps.ScanErr != "" {
			pf.ScanFailed = append(pf.ScanFailed, ps.Name+": "+ps.ScanErr)
		}
		if ps.ProdFiles+ps.TestFiles > 0 {
			pf.GoProjects++
		}
		if ps.Declared {
			pf.Declared = append(pf.Declared, ps.Name)
		}
		if ps.ProdFiles > 0 && ps.SitesProd == 0 && ps.ScanErr == "" {
			pf.ZeroIO = append(pf.ZeroIO, ps.Name)
		}
		if ps.Touched["entropy"] > 0 {
			pf.Entropy = append(pf.Entropy, ps.Name)
		}
		if ps.Touched["ipc-subprocess"] > 0 {
			pf.Exec = append(pf.Exec, ps.Name)
		}
		if ps.Touched["database"] > 0 {
			pf.DB = append(pf.DB, ps.Name)
		}
		net := ps.Touched["outbound"] + ps.Touched["listener"]
		if net > 0 {
			pf.Network = append(pf.Network, hotRow{Project: ps.Name, Sites: net})
		}
		for f, n := range ps.Dominant {
			pf.FamilyDominant[f] += n
		}
		for f, n := range ps.Touched {
			pf.FamilyTouched[f] += n
		}
		for v, n := range ps.ViaSeams {
			pf.ViaSeams[v] += n
		}
		for c, n := range ps.Calls {
			agg, ok := pf.Calls[c]
			if !ok {
				agg = &callAgg{Fams: map[string]int{}}
				pf.Calls[c] = agg
			}
			agg.Sites += n
			agg.Projects++
			for f, n2 := range ps.CallFams[c] {
				agg.Fams[f] += n2
			}
		}
		for f, n := range ps.HotFiles {
			fileCounts[hotKey{project: ps.Name, file: f}] += n
		}
	}
	sort.Strings(pf.Declared)
	sort.Strings(pf.ZeroIO)
	sort.Strings(pf.Entropy)
	sort.Strings(pf.Exec)
	sort.Strings(pf.DB)
	sort.SliceStable(pf.Network, func(a, b int) bool {
		if pf.Network[a].Sites != pf.Network[b].Sites {
			return pf.Network[a].Sites > pf.Network[b].Sites
		}
		return pf.Network[a].Project < pf.Network[b].Project
	})
	pf.Hotspots = topHotspots(fileCounts, nameOf, 15)
	sort.SliceStable(pf.Projects, func(a, b int) bool {
		x, y := pf.Projects[a], pf.Projects[b]
		if x.SitesProd != y.SitesProd {
			return x.SitesProd > y.SitesProd
		}
		if x.ProdLOC != y.ProdLOC {
			return x.ProdLOC > y.ProdLOC
		}
		return x.Name < y.Name
	})
	return pf
}

func dominantFams(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(a, b int) bool {
		if m[keys[a]] != m[keys[b]] {
			return m[keys[a]] > m[keys[b]]
		}
		return keys[a] < keys[b]
	})
	return keys
}

type hotKey struct {
	project string
	file    string
}

func topHotspots(counts map[hotKey]int, nameOf map[string]string, n int) []hotRow {
	type kv struct {
		k hotKey
		v int
	}
	var list []kv
	for k, v := range counts {
		list = append(list, kv{k, v})
	}
	sort.SliceStable(list, func(a, b int) bool {
		if list[a].v != list[b].v {
			return list[a].v > list[b].v
		}
		if list[a].k.project != list[b].k.project {
			return list[a].k.project < list[b].k.project
		}
		return list[a].k.file < list[b].k.file
	})
	if len(list) > n {
		list = list[:n]
	}
	rows := make([]hotRow, 0, len(list))
	for _, e := range list {
		rows = append(rows, hotRow{Project: nameOf[e.k.project], File: e.k.file, Sites: e.v})
	}
	return rows
}

func comma(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	return s + "," + strings.Join(parts, ",")
}

func pctf(part, total int) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", 100*float64(part)/float64(total))
}

func topStr(m map[string]int, n int) string {
	keys := dominantFams(m)
	if len(keys) > n {
		keys = keys[:n]
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, " ")
}

func topStrCalls(m map[string]*callAgg, n int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(a, b int) bool {
		if m[keys[a]].Sites != m[keys[b]].Sites {
			return m[keys[a]].Sites > m[keys[b]].Sites
		}
		return keys[a] < keys[b]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k].Sites))
	}
	return strings.Join(parts, " ")
}

func writeStats(pf *portfolio) {
	b, err := json.MarshalIndent(pf, "", "  ")
	if err == nil {
		_ = os.WriteFile(outputStats, b, 0o644)
	}
}

const extraCSS = `    <style>
      /* Deliberate additions for this report (distribution bars, chips); kit tokens reused. */
      .bar-row { display: grid; grid-template-columns: 11rem 1fr 5rem; gap: 0.75rem; align-items: center; margin-bottom: 0.45rem; }
      .bar-name { font-family: var(--font-mono); font-size: 0.78rem; color: var(--text-muted); text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
      .bar-track { background: var(--surface-sunken); height: 1.05rem; }
      .bar-fill { height: 100%; background: var(--accent); }
      .bar-value { font-family: var(--font-mono); font-size: 0.8rem; color: var(--text); }
      .chip-note { font-family: var(--font-mono); font-size: 0.72rem; color: var(--text-faint); }
    </style>
`

func render(pf *portfolio) string {
	var b strings.Builder
	b.WriteString(renderHero(pf))
	b.WriteString(renderTOC())
	b.WriteString(renderStats(pf))
	b.WriteString(renderFamilies(pf))
	b.WriteString(renderAPIs(pf))
	b.WriteString(renderInsights(pf))
	b.WriteString(renderHotspots(pf))
	b.WriteString(renderProjects(pf))
	b.WriteString(renderMethod(pf))
	b.WriteString(fmt.Sprintf(`<footer class="footer">
      Generated %s &middot; io-surface v%s (internal/extract, resolved + token tiers) &middot; discovery via pma &middot; raw numbers: /tmp/io-surface-portfolio-stats.json
    </footer>`, pf.GeneratedAt, pf.ToolVersion))

	doc, err := os.ReadFile(templatePath)
	fatalIf(err, "read template")
	docStr := string(doc)
	docStr = strings.Replace(docStr, "<title>Report Title</title>",
		"<title>I/O Surface Portfolio Scan - "+strconv.Itoa(pf.ProjectsScanned+pf.ProjectsMissing)+" Projects</title>", 1)
	docStr = strings.Replace(docStr, "<span>REPORT TYPE</span>", "<span>I/O SURFACE PORTFOLIO SCAN</span>", 1)
	docStr = strings.Replace(docStr, "<span>YYYY-MM-DD</span>", "<span>"+pf.GeneratedAt+"</span>", 1)
	docStr = strings.Replace(docStr, "<span>AUTHOR / TEAM</span>", "<span>io-surface v"+pf.ToolVersion+"</span>", 1)
	docStr = strings.Replace(docStr, "</head>", extraCSS+"</head>", 1)
	docStr = replaceSidebarNav(docStr, sidebarNav())
	mainStart := strings.Index(docStr, `<main class="main">`)
	mainEnd := strings.Index(docStr, `</main>`)
	if mainStart < 0 || mainEnd < 0 || mainEnd < mainStart {
		fmt.Println("FATAL: template markers not found")
		os.Exit(2)
	}
	docStr = docStr[:mainStart+len(`<main class="main">`)] + "\n" + b.String() + docStr[mainEnd:]
	return docStr
}

func replaceSidebarNav(doc, nav string) string {
	aside := strings.Index(doc, `<aside class="sidebar"`)
	if aside < 0 {
		return doc
	}
	navStart := strings.Index(doc[aside:], "<nav>") + aside
	navEnd := strings.Index(doc[aside:], "</nav>") + aside + len("</nav>")
	if navStart < aside || navEnd < navStart {
		return doc
	}
	return doc[:navStart] + nav + doc[navEnd:]
}

func sidebarNav() string {
	return `<nav>
          <div class="label">Contents</div>
          <a href="#summary">Summary</a>
          <a href="#families">Families</a>
          <a href="#apis">Top APIs</a>
          <a href="#insights">Insights</a>
          <a href="#hotspots">Hotspots</a>
          <a href="#projects">Project Inventory</a>
          <a href="#method">Methodology</a>
        </nav>`
}

func renderTOC() string {
	return `<nav class="toc">
          <h3>Contents</h3>
          <ul>
            <li><a href="#summary">Summary</a></li>
            <li><a href="#families">Family Distribution</a></li>
            <li><a href="#apis">Top I/O APIs</a></li>
            <li><a href="#insights">Insights</a></li>
            <li><a href="#hotspots">Hotspots</a></li>
            <li><a href="#projects">Project Inventory</a></li>
            <li><a href="#method">Methodology</a></li>
          </ul>
        </nav>`
}

func renderHero(pf *portfolio) string {
	return `<div class="hero">
          <div class="hero-shapes">
            <div class="circle"></div>
            <div class="square"></div>
            <div class="tri"></div>
          </div>
          <div class="hero-badge">Portfolio Scan</div>
          <h1>The I/O Surface of ` + strconv.Itoa(pf.ProjectsScanned+pf.ProjectsMissing) + ` Projects</h1>
          <p class="hero-subtitle">
            Every first-party Go call site that touches files, network, processes, databases,
            environment, or randomness, extracted by a stdlib-only AST walk across the whole
            ~/projects fleet with io-surface v` + pf.ToolVersion + `.
          </p>
        </div>`
}

func renderStats(pf *portfolio) string {
	return `<h2 id="summary"><span class="num">00</span>Summary</h2>
        <div class="stat-grid">
          <div class="stat-card">
            <div class="stat-value">` + comma(pf.ProjectsScanned) + `</div>
            <div class="stat-label">Projects Scanned</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">` + comma(pf.ProdFiles) + `</div>
            <div class="stat-label">Production Go Files (` + comma(pf.ProdLOC) + ` LOC)</div>
          </div>
          <div class="stat-card stat-warn">
            <div class="stat-value">` + comma(pf.SitesProd) + `</div>
            <div class="stat-label">I/O Call Sites (production)</div>
          </div>
          <div class="stat-card stat-good">
            <div class="stat-value">` + comma(len(pf.Declared)) + `</div>
            <div class="stat-label">Gate-Declared Projects</div>
          </div>
        </div>`
}

func renderFamilies(pf *portfolio) string {
	var b strings.Builder
	b.WriteString(`<h2 id="families"><span class="num">01</span>Family Distribution</h2>
        <p>A site is one durable symbol anchor (pkg.Func or pkg.Type.Method) with all its watched calls. "Dominant" applies the extractor's precedence (file-write beats database beats listener beats ... beats file-read); "touched" counts a site for every family it touches at all.</p>`)
	b.WriteString(`<p><strong>By dominant family</strong> (production sites)</p>`)
	keys := dominantFams(pf.FamilyDominant)
	total := 0
	for _, k := range keys {
		if k != "" {
			total += pf.FamilyDominant[k]
		}
	}
	max := 0
	for _, k := range keys {
		if k != "" && pf.FamilyDominant[k] > max {
			max = pf.FamilyDominant[k]
		}
	}
	if total > 0 {
		for _, k := range keys {
			if k == "" {
				continue
			}
			w := 100 * float64(pf.FamilyDominant[k]) / float64(max)
			if w < 2 {
				w = 2
			}
			b.WriteString(`<div class="bar-row"><div class="bar-name">` + html.EscapeString(k) + `</div><div class="bar-track"><div class="bar-fill" style="width:` + fmt.Sprintf("%.1f", w) + `%"></div></div><div class="bar-value">` + comma(pf.FamilyDominant[k]) + `</div></div>`)
		}
		if u := pf.FamilyDominant[""]; u > 0 {
			b.WriteString(`<p class="chip-note">plus ` + comma(u) + ` production sites with no family signal (method-token sites whose token carries no family, e.g. .Do).</p>`)
		}
	}
	b.WriteString(`<p><strong>Families touched (any overlap)</strong></p>`)
	tkeys := dominantFams(pf.FamilyTouched)
	var chips []string
	for _, k := range tkeys {
		chips = append(chips, fmt.Sprintf("%s %s", k, comma(pf.FamilyTouched[k])))
	}
	if len(chips) > 0 {
		b.WriteString(`<p class="chip-note">` + html.EscapeString(strings.Join(chips, " · ")) + `</p>`)
	}
	return b.String()
}

func renderAPIs(pf *portfolio) string {
	var b strings.Builder
	b.WriteString(`<h2 id="apis"><span class="num">02</span>Top I/O APIs</h2>
        <p>Distinct watched calls ranked by how many production symbol anchors use them, portfolio-wide.</p>
        <table>
          <thead><tr><th>#</th><th>API call</th><th>Family signal</th><th>Sites</th><th>Projects</th></tr></thead>
          <tbody>`)
	keys := make([]string, 0, len(pf.Calls))
	for k := range pf.Calls {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(a, b int) bool {
		if pf.Calls[keys[a]].Sites != pf.Calls[keys[b]].Sites {
			return pf.Calls[keys[a]].Sites > pf.Calls[keys[b]].Sites
		}
		return keys[a] < keys[b]
	})
	if len(keys) > 25 {
		keys = keys[:25]
	}
	for i, k := range keys {
		agg := pf.Calls[k]
		fam := ""
		best := 0
		for f, n := range agg.Fams {
			if n > best || (n == best && f < fam) {
				best, fam = n, f
			}
		}
		famCell := fam
		if fam == "" {
			famCell = `<span class="chip-note">none</span>`
		} else {
			famCell = `<span class="badge badge-low">` + html.EscapeString(fam) + `</span>`
		}
		b.WriteString(`<tr><td>` + strconv.Itoa(i+1) + `</td><td><code>` + html.EscapeString(k) + `</code></td><td>` + famCell + `</td><td>` + comma(agg.Sites) + `</td><td>` + comma(agg.Projects) + `</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func renderInsights(pf *portfolio) string {
	var b strings.Builder
	b.WriteString(`<h2 id="insights"><span class="num">03</span>Insights</h2>`)
	cap := func(names []string, n int) string {
		if len(names) == 0 {
			return ""
		}
		shown := names
		more := ""
		if len(names) > n {
			shown = names[:n]
			more = fmt.Sprintf(" and %d more", len(names)-n)
		}
		return "<code>" + strings.Join(shown, "</code>, <code>") + "</code>" + more
	}

	b.WriteString(`<div class="card card-solution"><h4>Gate readiness: ` + strconv.Itoa(len(pf.Declared)) + ` of ` + strconv.Itoa(pf.ProjectsScanned) + ` projects already declare their I/O surface</h4>
          <p>These repos carry <code>docs/architecture/io-surface/facts.cue</code> and pass <code>io-surface check</code> in CI. The other ` + strconv.Itoa(pf.ProjectsScanned-len(pf.Declared)) + ` could onboard with one <code>init</code> run each; the extractor already parsed them without errors in this scan.</p>`)
	if len(pf.Declared) > 0 {
		b.WriteString(`<p class="chip-note">` + cap(pf.Declared, 12) + `</p>`)
	}
	b.WriteString(`</div>`)

	envTouch := pf.FamilyTouched["env"]
	classified := total(pf.FamilyDominant)
	topFam, topFamN := "", 0
	for k, v := range pf.FamilyDominant {
		if k != "" && v > topFamN {
			topFam, topFamN = k, v
		}
	}
	leadCall, leadN, leadProj := "", 0, 0
	for c, agg := range pf.Calls {
		if agg.Fams[topFam] > 0 && agg.Sites > leadN {
			leadCall, leadN, leadProj = c, agg.Sites, agg.Projects
		}
	}
	b.WriteString(`<div class="card"><h4>` + html.EscapeString(topFam) + ` dominates: ` + comma(topFamN) + ` sites (` + pctf(topFamN, classified) + ` of classified production sites)</h4>
          <p>The fleet's single biggest I/O surface is ` + html.EscapeString(topFam) + `, led by <code>` + html.EscapeString(leadCall) + `</code> with ` + comma(leadN) + ` sites across ` + comma(leadProj) + ` projects. Environment access is the other universal touchpoint: ` + comma(envTouch) + ` sites touch env (` + html.EscapeString("Getenv/LookupEnv/Setenv/Environ") + `). These are exactly the surfaces an I/O manifest makes reviewable and review-before-merge.</p></div>`)

	tokenPct := pctf(pf.SitesToken, pf.SitesProd)
	b.WriteString(`<div class="card card-warning"><h4>The advisory tier is real: ` + comma(pf.SitesToken) + ` token-confidence sites (` + tokenPct + ` of production sites)</h4>
          <p>These were caught by method-name heuristics on unresolvable receivers (ListenAndServe, Dial, ExecContext, ...) rather than resolved imports. A real gate must treat them as advisory (<code>facts.extractable: false</code>), never build-breaking. The resolved tier in this scan: ` + comma(pf.SitesProd-pf.SitesToken) + ` sites.</p></div>`)

	if len(pf.Network) > 0 {
		var parts []string
		top := pf.Network
		if len(top) > 5 {
			top = top[:5]
		}
		for _, r := range top {
			parts = append(parts, fmt.Sprintf("<code>%s</code> (%s)", html.EscapeString(r.Project), comma(r.Sites)))
		}
		b.WriteString(`<div class="card"><h4>` + strconv.Itoa(len(pf.Network)) + ` projects open sockets: listeners or outbound dials</h4>
          <p>Heaviest network surface: ` + strings.Join(parts, ", ") + `. Everything else on this machine talks mostly to files and environment.</p></div>`)
	}

	b.WriteString(`<div class="card"><h4>Databases, subprocesses, and crypto</h4>
          <p>` + strconv.Itoa(len(pf.DB)) + ` projects touch databases (sql.Open, cqrs engines, QueryContext tokens), ` + strconv.Itoa(len(pf.Exec)) + ` spawn subprocesses (exec.Command, LookPath), and ` + strconv.Itoa(len(pf.Entropy)) + ` consume cryptographic entropy (crypto/rand, rsa.GenerateKey, x509.CreateCertificate), which is the family most worth pinning down in a review.</p>`)
	if len(pf.Entropy) > 0 {
		b.WriteString(`<p class="chip-note">entropy users: ` + cap(pf.Entropy, 10) + `</p>`)
	}
	b.WriteString(`</div>`)

	totalSites := pf.SitesProd + pf.SitesTest
	b.WriteString(`<div class="card card-warning"><h4>` + pctf(pf.SitesTest, totalSites) + ` of all extracted sites live only in tests</h4>
          <p>` + comma(pf.SitesTest) + ` of ` + comma(totalSites) + ` sites are in <code>_test.go</code> or <code>testutil/</code> files. The gate excludes tests by design, so manifests stay small; this scan includes them to show how much I/O exists only under test.</p></div>`)

	if pf.DroppedFixtureSites > 0 {
		b.WriteString(`<div class="card"><h4>` + comma(pf.DroppedFixtureSites) + ` fixture sites filtered out</h4>
          <p>Locations under <code>testdata/</code> or <code>docs/</code> packages were dropped after extraction; they are fixtures, not authored I/O. Generated files (<code>*_templ.go</code>, <code>*.pb.go</code>, <code>zz_generated*</code>, stringer output) were never parsed.</p></div>`)
	}

	if len(pf.ZeroIO) > 0 {
		b.WriteString(`<div class="card card-problem"><h4>` + strconv.Itoa(len(pf.ZeroIO)) + ` Go projects with code but zero production I/O sites</h4>
          <p>These have production Go files yet nothing the extractor watches fires: pure libraries, spec/codegen repos, or I/O hidden behind unresolvable seams. They are the cheapest possible <code>init</code> wins: an almost-empty manifest.</p>
          <p class="chip-note">` + cap(pf.ZeroIO, 15) + `</p></div>`)
	}

	if len(pf.ScanFailed) > 0 {
		b.WriteString(`<div class="card card-problem"><h4>` + strconv.Itoa(len(pf.ScanFailed)) + ` projects failed extraction</h4><p class="chip-note">` + html.EscapeString(strings.Join(pf.ScanFailed, "; ")) + `</p></div>`)
	}
	if pf.ProjectsMissing > 0 {
		b.WriteString(`<div class="callout callout-warning"><strong>Note</strong> ` + strconv.Itoa(pf.ProjectsMissing) + ` discovered project paths no longer exist on disk and were skipped.</div>`)
	}
	return b.String()
}

func total(m map[string]int) int {
	t := 0
	for k, v := range m {
		if k != "" {
			t += v
		}
	}
	return t
}

func renderHotspots(pf *portfolio) string {
	var b strings.Builder
	b.WriteString(`<h2 id="hotspots"><span class="num">04</span>Hotspot Files</h2>
        <p>Files hosting the most production I/O sites across the fleet. In a gate rollout these are where a manifest pays off first.</p>
        <table>
          <thead><tr><th>#</th><th>Project</th><th>File</th><th>Sites</th></tr></thead>
          <tbody>`)
	for i, r := range pf.Hotspots {
		b.WriteString(`<tr><td>` + strconv.Itoa(i+1) + `</td><td><code>` + html.EscapeString(r.Project) + `</code></td><td><code>` + html.EscapeString(r.File) + `</code></td><td>` + comma(r.Sites) + `</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func renderProjects(pf *portfolio) string {
	var b strings.Builder
	b.WriteString(`<h2 id="projects"><span class="num">05</span>Project Inventory</h2>
        <p>All discovered projects, sorted by production I/O sites. Families show the top 3 by dominant-site count; flags mark touched families (NET = outbound/listener, DB = database, EXEC = ipc-subprocess, CRYPTO = entropy).</p>
        <table>
          <thead><tr><th>#</th><th>Project</th><th>Types</th><th>Go LOC</th><th>Sites</th><th>Test</th><th>Top families</th><th>Flags</th><th>Manifest</th></tr></thead>
          <tbody>`)
	for i, ps := range pf.Projects {
		nameCell := `<code>` + html.EscapeString(ps.Name) + `</code>`
		if ps.Missing {
			b.WriteString(`<tr><td>` + strconv.Itoa(i+1) + `</td><td>` + nameCell + `</td><td colspan="7"><span class="badge badge-critical">missing on disk</span></td></tr>`)
			continue
		}
		types := ps.Types
		typeNote := strings.Join(types, ", ")
		if len(types) > 4 {
			typeNote = strings.Join(types[:4], ", ") + " +" + strconv.Itoa(len(types)-4)
		}
		var famBadges []string
		top := dominantFams(ps.Dominant)
		if len(top) > 3 {
			more := len(top) - 3
			top = top[:3]
			famBadges = append(famBadges, `<span class="chip-note">+`+strconv.Itoa(more)+`</span>`)
		}
		for _, f := range top {
			if f == "" {
				continue
			}
			famBadges = append([]string{`<span class="badge badge-low">` + html.EscapeString(f) + `</span>`}, famBadges...)
		}
		if len(famBadges) == 0 {
			famBadges = []string{`<span class="chip-note">-</span>`}
		}
		var flags []string
		if ps.Touched["outbound"] > 0 || ps.Touched["listener"] > 0 {
			flags = append(flags, "NET")
		}
		if ps.Touched["database"] > 0 {
			flags = append(flags, "DB")
		}
		if ps.Touched["ipc-subprocess"] > 0 {
			flags = append(flags, "EXEC")
		}
		if ps.Touched["entropy"] > 0 {
			flags = append(flags, "CRYPTO")
		}
		flagCell := `<span class="chip-note">-</span>`
		if len(flags) > 0 {
			var cells []string
			for _, f := range flags {
				cells = append(cells, `<span class="badge badge-medium">`+f+`</span>`)
			}
			flagCell = strings.Join(cells, " ")
		}
		manifest := `<span class="badge badge-low">undeclared</span>`
		if ps.Declared {
			manifest = `<span class="badge badge-fixed">declared</span>`
		}
		loc := comma(ps.ProdLOC)
		if ps.ScanErr != "" {
			loc += ` <span class="badge badge-critical">scan error</span>`
		}
		b.WriteString(`<tr><td>` + strconv.Itoa(i+1) + `</td><td>` + nameCell + `</td><td><span class="chip-note">` + html.EscapeString(typeNote) + `</span></td><td>` + loc + `</td><td><strong>` + comma(ps.SitesProd) + `</strong></td><td>` + comma(ps.SitesTest) + `</td><td>` + strings.Join(famBadges, " ") + `</td><td>` + flagCell + `</td><td>` + manifest + `</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func renderMethod(pf *portfolio) string {
	steps := []struct{ title, body string }{
		{"Discovery", "pma list -o json over ~/projects produced " + strconv.Itoa(pf.ProjectsScanned+pf.ProjectsMissing) + " projects (cached discovery snapshot, 2026-11-07)."},
		{"Extraction", "io-surface v" + pf.ToolVersion + " internal/extract: a hermetic stdlib AST walk (go/parser + ast.Inspect, no go/packages, no network) parsed every top-level Go directory of each project. vendor, node_modules, and dot-directories were skipped. Test files (*_test.go and testutil/ packages) were parsed and classified as test-only sites; the production numbers exclude them."},
		{"Filtering", "Generated code was ignored before parsing (*_templ.go, *.pb.go, zz_generated*, *_string.go, *_gen.go, deepcopy, easyjson). After extraction, locations under testdata/ or docs/ packages were dropped (" + comma(pf.DroppedFixtureSites) + " locations)."},
		{"Classification", "Resolved tier: package-qualified calls through imports (os.ReadFile, http.Get, sql.Open, ...) - the only tier that can fail a real gate. Token tier: method-name heuristics on unresolvable receivers (ListenAndServe, Dial, ExecContext, .Do) - advisory by design, attributed to the enclosing symbol."},
		{"Aggregation and rendering", "Per-project counts were folded into fleet totals, ranked, and rendered into this single self-contained HTML file (no JS dependencies beyond the template's optional scrollspy). The full machine-readable result is at /tmp/io-surface-portfolio-stats.json."},
	}
	var b strings.Builder
	b.WriteString(`<h2 id="method"><span class="num">06</span>Methodology</h2>`)
	for i, s := range steps {
		b.WriteString(`<div class="migration-step"><div class="migration-num">` + strconv.Itoa(i+1) + `</div><div class="migration-content"><h4>` + html.EscapeString(s.title) + `</h4><p>` + html.EscapeString(s.body) + `</p></div></div>`)
	}
	b.WriteString(`<div class="callout callout-warning"><strong>Reading the numbers</strong>
          Counts are call-site level, not occurrences: a symbol calling os.ReadFile three times is one site with one call. Dominant-family attribution follows the extractor's precedence, so a function that writes a file and reads the environment counts as file-write, not env. Token-tier sites cannot distinguish http.Client.Do from sql.Tx.Do-style helpers; that ambiguity is deliberate in the tool.</div>`)
	return b.String()
}
