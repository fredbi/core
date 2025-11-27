package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"golang.org/x/tools/benchmark/parse"
)

type cliFlags struct {
	File       string
	OutputFile string
	IsJSON     bool
}

func main() {
	var cli cliFlags

	flag.BoolVar(&cli.IsJSON, "json", false, "read input from JSON")
	flag.StringVar(&cli.File, "input", "-", "file input or - for standard input")
	flag.StringVar(&cli.OutputFile, "output", "-", "file output or - for standard output")

	flag.Parse()
	if err := execute(cli); err != nil {
		log.Fatalf("%v", err)
	}
}

func execute(c cliFlags) error {

	in, err := os.Open(c.File)
	if err != nil {
		return fmt.Errorf("input file: %w", err)
	}
	defer func() {
		_ = in.Close()
	}()

	b := New()
	b.input = in

	out, err := os.Create(c.OutputFile)
	if err != nil {
		return fmt.Errorf("output file: %w", err)
	}
	defer func() {
		if out != nil {
			_ = out.Close()
		}
	}()

	b.output = out

	if err := b.parseText(); err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if err := b.viz(); err != nil {
		return fmt.Errorf("viz: %w", err)
	}

	if err := out.Close(); err != nil {
		return err
	}
	out = nil

	reader, err := os.Open(c.OutputFile)
	if err != nil {
		return fmt.Errorf("reopen file: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	ext := path.Ext(c.OutputFile)
	image, _ := strings.CutSuffix(c.OutputFile, ext)
	if err := b.screenshot(image+".png", reader); err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}

	return nil
}

type inFormat uint8

const (
	inFormatText inFormat = iota
	inFormatJSON
)

type outFormat uint8

const (
	outFormatHTML inFormat = iota
	outFormatPNG
)

type BenchmarkVizualizer struct {
	options

	input  io.Reader
	output io.Writer
	set    parse.Set
}

type Option func(*options)

type options struct {
	inputOptions
	vizOptions
}

type inputOptions struct {
	inputFormat inFormat
}

type vizOptions struct {
	outputFormat outFormat
}

func New(opts ...Option) *BenchmarkVizualizer {
	return &BenchmarkVizualizer{}
}

func (b *BenchmarkVizualizer) parseText() error {
	set, err := parse.ParseSet(b.input)
	if err != nil {
		return err
	}

	b.set = set

	return nil
}

/*
	type Benchmark struct {
		Name              string  // benchmark name
		N                 int     // number of iterations
		NsPerOp           float64 // nanoseconds per iteration
		AllocedBytesPerOp uint64  // bytes allocated per iteration
		AllocsPerOp       uint64  // allocs per iteration
		MBPerS            float64 // MB processed per second
		Measured          int     // which measurements were recorded
		Ord               int     // ordinal position within a benchmark run
	}
*/
func onlyCategories(categories []string, subset ...string) []string {
	filtered := make([]string, 0, len(categories))
	for _, category := range categories {
		for _, allowed := range subset {
			if strings.Contains(category, allowed) {
				filtered = append(filtered, category)
			}
		}
	}
	return filtered
}

func filterSeries(data []opts.BarData, subset ...string) []opts.BarData {
	filtered := make([]opts.BarData, 0, len(data))
	for _, point := range data {
		categories := onlyCategories([]string{point.Name}, subset...)
		if len(categories) > 0 {
			filtered = append(filtered, point)
		}
	}
	return filtered
}

func categorize(bench *parse.Benchmark) string {
	s := bench.Name
	var w strings.Builder

	switch {
	case strings.Contains(s, "ReadJSON"):
		w.WriteString("ReadJSON")
	case strings.Contains(s, "WriteJSON"):
		w.WriteString("WriteJSON")
	case strings.Contains(s, "FromDynamicJSON"):
		w.WriteString("FromDynamicJSON")
	}

	w.WriteString(" - ")
	switch {
	case strings.Contains(s, "small"):
		w.WriteString("small")
	case strings.Contains(s, "medium"):
		w.WriteString("medium")
	case strings.Contains(s, "large"):
		w.WriteString("large")
	}

	return w.String()
}

func categoryOrder(category string) int {
	s := category
	var idx int

	switch {
	case strings.Contains(s, "ReadJSON"):
		idx = 0
	case strings.Contains(s, "WriteJSON"):
		idx = 4
	case strings.Contains(s, "FromDynamicJSON"):
		idx = 8
	}

	switch {
	case strings.Contains(s, "small"):
		idx |= 0
	case strings.Contains(s, "medium"):
		idx |= 1
	case strings.Contains(s, "large"):
		idx |= 2
	}

	return idx
}

func (b *BenchmarkVizualizer) viz() error {
	const environment = "go1.25.0 linux/amd64 cpu: AMD Ryzen 7 5800X 8-Core Processor"

	series1 := make([]*parse.Benchmark, 0, len(b.set))
	series2 := make([]*parse.Benchmark, 0, len(b.set))
	for _, benchs := range b.set {
		for _, bench := range benchs {
			if strings.Contains(bench.Name, "easyjson") {
				series2 = append(series2, bench)
				continue
			}
			series1 = append(series1, bench)
		}
	}

	categories := make([]string, 0, len(series1))
	categoryIndex := make(map[string]struct{}, len(series1))

	for _, bench := range series1 {
		category := categorize(bench)
		_, ok := categoryIndex[category]
		if ok {
			continue
		}
		categories = append(categories, category)
		categoryIndex[category] = struct{}{}
	}

	sort.Slice(categories, func(i, j int) bool {
		return categoryOrder(categories[i]) < categoryOrder(categories[j])
	})

	sort.Slice(series1, func(i, j int) bool {
		return categoryOrder(categorize(series1[i])) < categoryOrder(categorize(series1[j]))
	})
	sort.Slice(series2, func(i, j int) bool {
		return categoryOrder(categorize(series2[i])) < categoryOrder(categorize(series2[j]))
	})

	ns1 := make([]opts.BarData, 0, len(series1))
	for _, bench := range series1 {
		ns1 = append(ns1, opts.BarData{
			Name:  bench.Name,
			Value: int(math.Round(bench.NsPerOp)),
		})
	}
	ns2 := make([]opts.BarData, 0, len(series2))
	for _, bench := range series2 {
		ns2 = append(ns2, opts.BarData{
			Name:  bench.Name,
			Value: int(math.Round(bench.NsPerOp)),
		})
	}

	allocs1 := make([]opts.BarData, 0, len(series1))
	for _, bench := range series1 {
		allocs1 = append(allocs1, opts.BarData{
			Name:  bench.Name,
			Value: bench.AllocsPerOp,
		})
	}
	allocs2 := make([]opts.BarData, 0, len(series2))
	for _, bench := range series2 {
		allocs2 = append(allocs2, opts.BarData{
			Name:  bench.Name,
			Value: bench.AllocsPerOp,
		})
	}

	xAxis := opts.XAxis{
		Name:         "Workload",
		Type:         "category",
		Position:     "bottom",
		NameLocation: "end",
		AxisTick: &opts.AxisTick{
			AlignWithLabel: opts.Bool(true),
		},
		AxisLabel: &opts.AxisLabel{
			Rotate:       30,
			Interval:     "0",
			ShowMinLabel: opts.Bool(true),
			ShowMaxLabel: opts.Bool(true),
			HideOverlap:  opts.Bool(false),
		},
	}

	yAxisTimings := opts.YAxis{
		Name:  "ns/op",
		Type:  "value",
		Scale: opts.Bool(true),
		AxisLabel: &opts.AxisLabel{
			Formatter: opts.FuncOpts("function (value,index) { return value.toFixed(0).toString();}"),
		},
	}

	yAxisAllocs := opts.YAxis{
		Name:  "allocs/op",
		Type:  "value",
		Scale: opts.Bool(true),
		AxisLabel: &opts.AxisLabel{
			Formatter: opts.FuncOpts("function (value,index) { return value.toFixed(0).toString();}"),
		},
	}

	grid := opts.Grid{
		Bottom: "100",
		Top:    "100",
	}

	toolbox := opts.Toolbox{
		Left: "right",
		Feature: &opts.ToolBoxFeature{
			SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
				Title: "Save as image",
			},
		},
	}

	smallAndMedium := charts.NewBar()
	smallAndMedium.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeRoma}),
		charts.WithToolboxOpts(toolbox),
		charts.WithTitleOpts(
			opts.Title{Title: "ReadJSON/WriteJSON timings",
				Subtitle: environment,
				SubtitleStyle: &opts.TextStyle{
					FontStyle: "italic",
					FontSize:  12,
				},
			},
		),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
		charts.WithGridOpts(grid),
		charts.WithXAxisOpts(xAxis),
		charts.WithYAxisOpts(yAxisTimings),
	)
	smallAndMedium.SetXAxis(onlyCategories(categories, "small", "medium"))

	large := charts.NewBar()
	large.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeRoma}),
		charts.WithToolboxOpts(toolbox),
		charts.WithTitleOpts(opts.Title{
			Title:    "ReadJSON/WriteJSON timings (large)",
			Subtitle: environment,
			SubtitleStyle: &opts.TextStyle{
				FontStyle: "italic",
				FontSize:  12,
			},
		},
		),
		charts.WithLegendOpts(opts.Legend{
			X: "right",
			Y: "bottom",
		}),
		charts.WithGridOpts(grid),
		charts.WithXAxisOpts(xAxis),
		charts.WithYAxisOpts(yAxisTimings),
	)
	large.SetXAxis(onlyCategories(categories, "large"))

	smallAndMedium.AddSeries("stdlib", filterSeries(ns1, "small", "medium"))
	smallAndMedium.AddSeries("easyjson", filterSeries(ns2, "small", "medium"))

	large.AddSeries("stdlib", filterSeries(ns1, "large"))
	large.AddSeries("easyjson", filterSeries(ns2, "large"))

	smallAndMediumAllocs := charts.NewBar()
	smallAndMediumAllocs.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeRoma}),
		charts.WithToolboxOpts(toolbox),
		charts.WithTitleOpts(
			opts.Title{Title: "ReadJSON/WriteJSON allocs"},
		),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(false),
		}),
		charts.WithGridOpts(grid),
		charts.WithXAxisOpts(xAxis),
		charts.WithYAxisOpts(yAxisAllocs),
	)
	smallAndMediumAllocs.SetXAxis(onlyCategories(categories, "small", "medium"))

	largeAllocs := charts.NewBar()
	largeAllocs.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeRoma}),
		charts.WithToolboxOpts(toolbox),
		charts.WithTitleOpts(opts.Title{
			Title: "ReadJSON/WriteJSON allocs (large)"},
		),
		charts.WithLegendOpts(opts.Legend{
			X: "right",
			Y: "bottom",
		}),
		charts.WithGridOpts(grid),
		charts.WithXAxisOpts(xAxis),
		charts.WithYAxisOpts(yAxisAllocs),
	)
	largeAllocs.SetXAxis(onlyCategories(categories, "large"))

	smallAndMediumAllocs.AddSeries("stdlib", filterSeries(allocs1, "small", "medium"))
	smallAndMediumAllocs.AddSeries("easyjson", filterSeries(allocs2, "small", "medium"))

	largeAllocs.AddSeries("stdlib", filterSeries(allocs1, "large"))
	largeAllocs.AddSeries("easyjson", filterSeries(allocs2, "large"))

	page := components.NewPage()
	page.SetLayout(components.PageFlexLayout)
	page.SetPageTitle(environment)

	page.AddCharts(smallAndMedium, large, smallAndMediumAllocs, largeAllocs)

	if err := page.Render(b.output); err != nil {
		return err
	}
	return nil
}

func (b *BenchmarkVizualizer) screenshot(name string, r io.Reader) error {
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		// chromedp.WithDebugf(log.Printf),
		// chromedp.WithBrowserOption(opts ...chromedp.BrowserOption)
	)
	defer cancel()
	var screenshot []byte
	// capture entire browser viewport, returning png with quality=90
	// localURL := fmt.Sprintf(`file://./%s`, file)
	content, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read content: %w", err)
	}

	if err := chromedp.Run(ctx,
		chromedp.Emulate(device.Info{
			Height:    1080,
			Width:     1920,
			Landscape: true,
		}),
		chromedp.Navigate("data:text/html,"+string(content)),
		// chromedp.WaitVisible(`canvas`, chromedp.ByQueryAll),
		// chromedp.WaitReady(`script  _, opts ...chromedp.QueryOption),
		chromedp.Sleep(time.Second),
		chromedp.FullScreenshot(&screenshot, 100),
	); err != nil {
		return err
	}

	if err := os.WriteFile(name, screenshot, 0o644); err != nil {
		return err
	}
	return nil
}
