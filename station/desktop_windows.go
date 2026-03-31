//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/sys/windows"
)

func maybeRunDesktopUI() bool {
	if os.Getenv("STATION_DISABLE_DESKTOP_UI") != "" {
		return false
	}
	forceDesktop := os.Getenv("STATION_FORCE_DESKTOP_UI") != ""
	if !forceDesktop {
		parentName, err := parentProcessName()
		if err != nil || !shouldLaunchDesktopUI(os.Args, parentName) {
			return false
		}
	}
	if err := runDesktopUI(); err != nil {
		showDesktopError(err)
	}
	return true
}

func parentProcessName() (string, error) {
	ppid := windows.Getppid()
	if ppid <= 0 {
		return "", fmt.Errorf("parent pid unavailable")
	}
	return processNameByID(uint32(ppid))
}

func processNameByID(pid uint32) (string, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return "", err
	}
	for {
		if entry.ProcessID == pid {
			return windows.UTF16ToString(entry.ExeFile[:]), nil
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			return "", err
		}
	}
}

// ─── row widget-state types ───────────────────────────────────────────────────

type containerRow struct {
	data *ContainerRecord
	stop widget.Clickable
	logs widget.Clickable
}

type buildRow struct {
	id      string
	app     string
	started time.Time
	logs    widget.Clickable
}

type imageRow struct {
	name string
	rm   widget.Clickable
}

type portRow struct {
	app  string
	port int
	free widget.Clickable
}

type snapshotRow struct {
	name string
	load widget.Clickable
	del  widget.Clickable
}

type proxyRow struct {
	data *SlotRecord
	swap widget.Clickable
	stop widget.Clickable
}

// ─── dialog / modal state ─────────────────────────────────────────────────────

type dialogKind int

const (
	dialogNone dialogKind = iota
	dialogRun
	dialogProxySwap
	dialogSnapshotLoad
	dialogSetupRootfs
)

type dialogState struct {
	kind      dialogKind
	title     string
	confirm   widget.Clickable
	cancel    widget.Clickable
	field1    widget.Editor // primary text input
	field2    widget.Editor // secondary text input (if needed)
	label1    string
	label2    string
	contextID string // app name / snapshot name / etc.
}

// ─── application state ───────────────────────────────────────────────────────

type stationUI struct {
	mu sync.Mutex

	activeTab  int
	tabBtns    [6]widget.Clickable
	refreshBtn widget.Clickable
	setupBtn   widget.Clickable
	wslWarmBtn widget.Clickable

	containers []containerRow
	builds     []buildRow
	images     []imageRow
	ports      []portRow
	snapshots  []snapshotRow
	proxies    []proxyRow

	cList layout.List
	bList layout.List
	iList layout.List
	pList layout.List
	sList layout.List
	xList layout.List

	dialog dialogState

	statusMsg   string
	statusOk    bool
	lastRefresh time.Time
	loading     bool
}

// ─── colour palette ──────────────────────────────────────────────────────────

var (
	// Header / chrome
	clrHdr    = color.NRGBA{R: 15, G: 20, B: 30, A: 255}
	clrTabBar = color.NRGBA{R: 22, G: 30, B: 45, A: 255}
	clrTabOn  = color.NRGBA{R: 38, G: 55, B: 80, A: 255}
	clrTabTxt = color.NRGBA{R: 185, G: 205, B: 228, A: 255}

	// Table
	clrHead = color.NRGBA{R: 240, G: 243, B: 248, A: 255}
	clrEven = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	clrOdd  = color.NRGBA{R: 248, G: 250, B: 253, A: 255}

	// Text
	clrDark  = color.NRGBA{R: 20, G: 28, B: 40, A: 255}
	clrMuted = color.NRGBA{R: 110, G: 126, B: 148, A: 255}
	clrWhite = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// Status bar
	clrSbar = color.NRGBA{R: 235, G: 240, B: 248, A: 255}

	// Buttons
	clrBtnRed    = color.NRGBA{R: 185, G: 38, B: 50, A: 255}
	clrBtnBlue   = color.NRGBA{R: 28, G: 88, B: 200, A: 255}
	clrBtnOrange = color.NRGBA{R: 200, G: 100, B: 18, A: 255}
	clrBtnGray   = color.NRGBA{R: 80, G: 95, B: 115, A: 255}
	clrBtnGreen  = color.NRGBA{R: 18, G: 145, B: 72, A: 255}
	clrBtnIndigo = color.NRGBA{R: 68, G: 60, B: 195, A: 255}

	// Semantic
	clrGreen = color.NRGBA{R: 18, G: 145, B: 72, A: 255}

	// Modal overlay
	clrOverlay = color.NRGBA{R: 10, G: 15, B: 25, A: 200}
	clrModal   = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	clrInput   = color.NRGBA{R: 245, G: 247, B: 252, A: 255}
	clrBorder  = color.NRGBA{R: 210, G: 218, B: 232, A: 255}

	// Status dot
	clrDotOk  = color.NRGBA{R: 18, G: 160, B: 80, A: 255}
	clrDotErr = color.NRGBA{R: 190, G: 40, B: 50, A: 255}
)

// ─── infrastructure ───────────────────────────────────────────────────────────

func hideConsoleWindow() {}

func showDesktopError(err error) {
	msg := fmt.Sprintf("Station Desktop failed to start.\n\nError: %v\n\nLog: %s", err, desktopErrorLogPath())
	_, _ = fmt.Fprintf(os.Stderr, "Desktop error: %v\n", err)
	_ = os.WriteFile(desktopErrorLogPath(), []byte(msg+"\n"), 0644)
}

func runDesktopUI() error {
	errCh := make(chan error, 1)
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Station Desktop"))
		w.Option(app.Size(unit.Dp(1100), unit.Dp(720)))
		errCh <- runGioDesktopWindow(w)
		os.Exit(0)
	}()
	app.Main()
	return <-errCh
}

func desktopErrorLogPath() string {
	return filepath.Join(os.TempDir(), "station-desktop-error.log")
}

// ─── data loading ────────────────────────────────────────────────────────────

func loadAllData(st *stationUI) {
	// Containers
	recs := allRecords()
	sort.Slice(recs, func(i, j int) bool { return recs[i].Started.After(recs[j].Started) })
	ctrs := make([]containerRow, len(recs))
	for i, r := range recs {
		ctrs[i] = containerRow{data: r}
	}

	// Builds – read from build log dir if available
	var blds []buildRow
	if entries, err := os.ReadDir(buildLogDir()); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			info, _ := e.Info()
			blds = append(blds, buildRow{
				id:      e.Name(),
				app:     buildAppName(e.Name()),
				started: safeModTime(info),
			})
		}
		sort.Slice(blds, func(i, j int) bool { return blds[i].started.After(blds[j].started) })
	}

	// Images
	var imgs []imageRow
	if entries, err := os.ReadDir(ociCacheDir()); err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != "layers" {
				name := strings.NewReplacer("__", ":", "_", "/").Replace(e.Name())
				imgs = append(imgs, imageRow{name: name})
			}
		}
	}

	// Ports
	pm := loadPortState()
	pApps := make([]string, 0, len(pm))
	for a := range pm {
		pApps = append(pApps, a)
	}
	sort.Strings(pApps)
	pts := make([]portRow, len(pApps))
	for i, a := range pApps {
		pts[i] = portRow{app: a, port: pm[a]}
	}

	// Snapshots
	var snaps []snapshotRow
	if entries, err := os.ReadDir(snapshotStore()); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				snaps = append(snaps, snapshotRow{name: e.Name()})
			}
		}
	}

	// Proxies
	slots := allSlotRecords()
	prxs := make([]proxyRow, len(slots))
	for i, r := range slots {
		prxs[i] = proxyRow{data: r}
	}

	st.mu.Lock()
	st.containers = ctrs
	st.builds = blds
	st.images = imgs
	st.ports = pts
	st.snapshots = snaps
	st.proxies = prxs
	st.lastRefresh = time.Now()
	st.loading = false
	st.mu.Unlock()
}

// buildLogDir returns the directory where build logs are stored.
func buildLogDir() string {
	return filepath.Join(os.TempDir(), "station-builds")
}

// buildAppName extracts a human-readable app name from a build ID.
func buildAppName(id string) string {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return id
}

func safeModTime(info os.FileInfo) time.Time {
	if info == nil {
		return time.Time{}
	}
	return info.ModTime()
}

// ─── actions ─────────────────────────────────────────────────────────────────

func (st *stationUI) runAction(w *app.Window, args ...string) {
	exe, err := os.Executable()
	if err != nil {
		st.mu.Lock()
		st.statusMsg = "error: cannot resolve executable"
		st.statusOk = false
		st.mu.Unlock()
		return
	}
	st.mu.Lock()
	st.statusMsg = "● station " + strings.Join(args, " ")
	st.statusOk = true
	st.mu.Unlock()
	w.Invalidate()
	go func() {
		out, err := exec.Command(exe, args...).CombinedOutput()
		msg := strings.TrimSpace(string(out))
		if err != nil && msg == "" {
			msg = err.Error()
		}
		if len(msg) > 140 {
			msg = msg[:140] + "…"
		}
		st.mu.Lock()
		st.statusMsg = msg
		st.statusOk = err == nil
		st.mu.Unlock()
		loadAllData(st)
		w.Invalidate()
	}()
}

func (st *stationUI) openLogsWindow(id string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command("powershell.exe", "-NoExit", "-Command",
		fmt.Sprintf("& '%s' logs '%s'", exe, id))
	_ = cmd.Start()
}

func (st *stationUI) openBuildLogs(buildID string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command("powershell.exe", "-NoExit", "-Command",
		fmt.Sprintf("& '%s' build-logs '%s'", exe, buildID))
	_ = cmd.Start()
}

// ─── dialog helpers ───────────────────────────────────────────────────────────

func (st *stationUI) openDialog(kind dialogKind, title, label1, label2, hint1, hint2, contextID string) {
	st.dialog = dialogState{
		kind:      kind,
		title:     title,
		label1:    label1,
		label2:    label2,
		contextID: contextID,
	}
	st.dialog.field1.SetText(hint1)
	st.dialog.field2.SetText(hint2)
	st.dialog.field1.SingleLine = true
	st.dialog.field2.SingleLine = true
}

func (st *stationUI) closeDialog() {
	st.dialog.kind = dialogNone
}

func (st *stationUI) submitDialog(w *app.Window) {
	d := &st.dialog
	v1 := strings.TrimSpace(d.field1.Text())
	v2 := strings.TrimSpace(d.field2.Text())
	switch d.kind {
	case dialogProxySwap:
		if v1 != "" {
			st.runAction(w, "proxy", "swap", "--app", d.contextID, "--upstream", v1)
		}
	case dialogSnapshotLoad:
		if v1 != "" {
			st.runAction(w, "snapshot", "load", d.contextID, v1)
		}
	case dialogRun:
		// v1 = dir, v2 = cmd
		args := []string{"run"}
		if d.contextID != "" {
			args = append(args, "--app", d.contextID)
		}
		if v1 != "" {
			args = append(args, v1)
		}
		if v2 != "" {
			args = append(args, strings.Fields(v2)...)
		}
		if len(args) > 1 {
			st.runAction(w, args...)
		}
	case dialogSetupRootfs:
		args := []string{"setup-rootfs"}
		if v1 != "" {
			args = append(args, v1)
		}
		st.runAction(w, args...)
	}
	st.closeDialog()
}

// ─── layout primitives ───────────────────────────────────────────────────────

func withBg(gtx layout.Context, c color.NRGBA, w layout.Widget) layout.Dimensions {
	paint.FillShape(gtx.Ops, c, clip.Rect{Max: gtx.Constraints.Max}.Op())
	return w(gtx)
}

func rect(gtx layout.Context, c color.NRGBA, r image.Rectangle) {
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

func cell(th *material.Theme, sp unit.Sp, txt string, c color.NRGBA) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Left: unit.Dp(12), Right: unit.Dp(4),
			Top: unit.Dp(13), Bottom: unit.Dp(13),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(th, sp, txt)
			l.Color = c
			l.MaxLines = 1
			return l.Layout(gtx)
		})
	}
}

func headCell(th *material.Theme, txt string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Left: unit.Dp(12), Right: unit.Dp(4),
			Top: unit.Dp(7), Bottom: unit.Dp(7),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			l := material.Label(th, unit.Sp(10), strings.ToUpper(txt))
			l.Color = clrMuted
			l.Font.Weight = font.Bold
			l.MaxLines = 1
			return l.Layout(gtx)
		})
	}
}

func btn(gtx layout.Context, th *material.Theme, b *widget.Clickable, label string, bg color.NRGBA) layout.Dimensions {
	return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, bg, clip.RRect{
					Rect: image.Rectangle{Max: gtx.Constraints.Min},
					NE:   4, NW: 4, SE: 4, SW: 4,
				}.Op(gtx.Ops))
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Left: unit.Dp(11), Right: unit.Dp(11),
					Top: unit.Dp(5), Bottom: unit.Dp(5),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, unit.Sp(12), label)
					l.Color = clrWhite
					l.Font.Weight = font.Bold
					return l.Layout(gtx)
				})
			}),
		)
	})
}

func tabButton(gtx layout.Context, th *material.Theme, b *widget.Clickable, label string, active bool) layout.Dimensions {
	return b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		bg := clrTabBar
		if active {
			bg = clrTabOn
		}
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Min}.Op())
				if active {
					// Active indicator line at bottom
					h := gtx.Constraints.Min.Y
					accentLine := image.Rectangle{
						Min: image.Pt(0, h-3),
						Max: image.Pt(gtx.Constraints.Min.X, h),
					}
					paint.FillShape(gtx.Ops, clrBtnBlue, clip.Rect(accentLine).Op())
				}
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Left: unit.Dp(18), Right: unit.Dp(18),
					Top: unit.Dp(10), Bottom: unit.Dp(13),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, unit.Sp(13), label)
					l.Color = clrTabTxt
					if active {
						l.Font.Weight = font.Bold
					}
					return l.Layout(gtx)
				})
			}),
		)
	})
}

func emptyMsg(th *material.Theme, icon, msg, hint string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, unit.Sp(28), icon)
					l.Color = clrBorder
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, unit.Sp(15), msg)
					l.Color = clrDark
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, unit.Sp(12), hint)
					l.Color = clrMuted
					return l.Layout(gtx)
				}),
			)
		})
	}
}

func divider(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(1)
	paint.FillShape(gtx.Ops, clrBorder, clip.Rect{
		Max: image.Pt(gtx.Constraints.Max.X, h),
	}.Op())
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, h)}
}

func numStr(n int) string { return fmt.Sprintf("%d", n) }

func statusDot(gtx layout.Context, ok bool) layout.Dimensions {
	c := clrDotErr
	if ok {
		c = clrDotOk
	}
	size := gtx.Dp(7)
	defer clip.RRect{
		Rect: image.Rectangle{Max: image.Pt(size, size)},
		NE:   size / 2, NW: size / 2, SE: size / 2, SW: size / 2,
	}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, c)
	return layout.Dimensions{Size: image.Pt(size, size)}
}

// ─── modal dialog ────────────────────────────────────────────────────────────

func (st *stationUI) drawModal(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	d := &st.dialog
	if d.kind == dialogNone {
		return layout.Dimensions{}
	}

	// Check confirm / cancel
	for d.confirm.Clicked(gtx) {
		st.submitDialog(w)
	}
	for d.cancel.Clicked(gtx) {
		st.closeDialog()
	}

	// Full-screen dim overlay
	paint.FillShape(gtx.Ops, clrOverlay, clip.Rect{Max: gtx.Constraints.Max}.Op())

	// Modal card, centred
	modalW := gtx.Dp(440)
	x := (gtx.Constraints.Max.X - modalW) / 2
	y := (gtx.Constraints.Max.Y - gtx.Dp(260)) / 2
	if y < 0 {
		y = 0
	}

	defer op.Offset(image.Pt(x, y)).Push(gtx.Ops).Pop()
	gtx.Constraints.Max.X = modalW
	gtx.Constraints.Min.X = modalW

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, clrModal, clip.RRect{
				Rect: image.Rectangle{Max: image.Pt(modalW, gtx.Dp(300))},
				NE:   8, NW: 8, SE: 8, SW: 8,
			}.Op(gtx.Ops))
			return layout.Dimensions{Size: image.Pt(modalW, gtx.Dp(300))}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					// Title
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(th, unit.Sp(16), d.title)
						l.Color = clrDark
						l.Font.Weight = font.Bold
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					// Field 1
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(th, unit.Sp(11), strings.ToUpper(d.label1))
								l.Color = clrMuted
								l.Font.Weight = font.Bold
								return l.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								e := material.Editor(th, &d.field1, "")
								e.Color = clrDark
								e.HintColor = clrMuted
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
										paint.FillShape(gtx.Ops, clrInput, clip.RRect{
											Rect: image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(36))},
											NE:   4, NW: 4, SE: 4, SW: 4,
										}.Op(gtx.Ops))
										return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(36))}
									}),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10), Top: unit.Dp(9), Bottom: unit.Dp(9)}.Layout(gtx, e.Layout)
									}),
								)
							}),
						)
					}),
					// Field 2 (optional)
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if d.label2 == "" {
							return layout.Dimensions{}
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(th, unit.Sp(11), strings.ToUpper(d.label2))
								l.Color = clrMuted
								l.Font.Weight = font.Bold
								return l.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								e := material.Editor(th, &d.field2, "")
								e.Color = clrDark
								e.HintColor = clrMuted
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(func(gtx layout.Context) layout.Dimensions {
										paint.FillShape(gtx.Ops, clrInput, clip.RRect{
											Rect: image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(36))},
											NE:   4, NW: 4, SE: 4, SW: 4,
										}.Op(gtx.Ops))
										return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(36))}
									}),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.Inset{Left: unit.Dp(10), Right: unit.Dp(10), Top: unit.Dp(9), Bottom: unit.Dp(9)}.Layout(gtx, e.Layout)
									}),
								)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(22)}.Layout),
					// Buttons
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return btn(gtx, th, &d.cancel, "Cancel", clrBtnGray)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return btn(gtx, th, &d.confirm, "Confirm", clrBtnBlue)
							}),
						)
					}),
				)
			})
		}),
	)
}

// ─── main window loop ────────────────────────────────────────────────────────

func runGioDesktopWindow(w *app.Window) error {
	th := material.NewTheme()
	var ops op.Ops
	st := &stationUI{
		cList:     layout.List{Axis: layout.Vertical},
		bList:     layout.List{Axis: layout.Vertical},
		iList:     layout.List{Axis: layout.Vertical},
		pList:     layout.List{Axis: layout.Vertical},
		sList:     layout.List{Axis: layout.Vertical},
		xList:     layout.List{Axis: layout.Vertical},
		statusMsg: "Loading data…",
		statusOk:  true,
	}

	go func() {
		loadAllData(st)
		w.Invalidate()
	}()

	go func() {
		tick := time.NewTicker(5 * time.Second)
		for range tick.C {
			st.mu.Lock()
			busy := st.loading
			st.mu.Unlock()
			if !busy {
				st.mu.Lock()
				st.loading = true
				st.mu.Unlock()
				go func() { loadAllData(st); w.Invalidate() }()
			}
		}
	}()

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			st.mu.Lock()
			// Tab clicks
			for i := range st.tabBtns {
				for st.tabBtns[i].Clicked(gtx) {
					st.activeTab = i
				}
			}
			// Refresh
			for st.refreshBtn.Clicked(gtx) {
				if !st.loading {
					st.loading = true
					st.statusMsg = "Refreshing…"
					st.statusOk = true
					go func() { loadAllData(st); w.Invalidate() }()
				}
			}
			// Setup rootfs
			for st.setupBtn.Clicked(gtx) {
				st.openDialog(dialogSetupRootfs, "Setup Rootfs", "Destination path (optional)", "", "", "", "")
			}
			// WSL warm
			for st.wslWarmBtn.Clicked(gtx) {
				go st.runAction(w, "wsl-warm")
			}

			activeTab := st.activeTab
			counts := [6]int{
				len(st.containers),
				len(st.builds),
				len(st.images),
				len(st.ports),
				len(st.snapshots),
				len(st.proxies),
			}
			lastRefresh := st.lastRefresh
			statusMsg := st.statusMsg
			statusOk := st.statusOk
			dlgKind := st.dialog.kind
			st.mu.Unlock()

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// ── header ───────────────────────────────────────────────────
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return withBg(gtx, clrHdr, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Left: unit.Dp(20), Right: unit.Dp(14),
							Top: unit.Dp(12), Bottom: unit.Dp(12),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								// Logo / title
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											l := material.Label(th, unit.Sp(18), "Station Desktop")
											l.Color = clrWhite
											l.Font.Weight = font.Bold
											return l.Layout(gtx)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											l := material.Label(th, unit.Sp(10), "Container runtime · WSL2")
											l.Color = clrMuted
											return l.Layout(gtx)
										}),
									)
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
								// Utility buttons
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return btn(gtx, th, &st.setupBtn, "Setup Rootfs", clrBtnGray)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return btn(gtx, th, &st.wslWarmBtn, "WSL Warm", clrBtnGray)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
								// Refresh + last updated
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if lastRefresh.IsZero() {
										return layout.Dimensions{}
									}
									l := material.Label(th, unit.Sp(11), lastRefresh.Format("15:04:05"))
									l.Color = clrMuted
									return l.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return btn(gtx, th, &st.refreshBtn, "↻  Refresh", clrBtnBlue)
								}),
							)
						})
					})
				}),
				// ── tab strip ────────────────────────────────────────────────
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return withBg(gtx, clrTabBar, func(gtx layout.Context) layout.Dimensions {
						names := [6]string{"Containers", "Builds", "Images", "Ports", "Snapshots", "Proxies"}
						children := make([]layout.FlexChild, 6)
						for i := range names {
							i := i
							label := fmt.Sprintf("%s  %d", names[i], counts[i])
							children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return tabButton(gtx, th, &st.tabBtns[i], label, activeTab == i)
							})
						}
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
					})
				}),
				// ── content ──────────────────────────────────────────────────
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					// Draw tab content in background, modal on top
					return layout.Stack{}.Layout(gtx,
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							switch activeTab {
							case 0:
								return st.drawContainersTab(gtx, th, w)
							case 1:
								return st.drawBuildsTab(gtx, th, w)
							case 2:
								return st.drawImagesTab(gtx, th, w)
							case 3:
								return st.drawPortsTab(gtx, th, w)
							case 4:
								return st.drawSnapshotsTab(gtx, th, w)
							default:
								return st.drawProxiesTab(gtx, th, w)
							}
						}),
						// Modal overlay (no-op when dialogNone)
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							if dlgKind == dialogNone {
								return layout.Dimensions{}
							}
							return st.drawModal(gtx, th, w)
						}),
					)
				}),
				// ── status bar ───────────────────────────────────────────────
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return withBg(gtx, clrSbar, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Left: unit.Dp(12), Right: unit.Dp(16),
							Top: unit.Dp(5), Bottom: unit.Dp(5),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Right: unit.Dp(7), Top: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return statusDot(gtx, statusOk)
									})
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(th, unit.Sp(12), statusMsg)
									l.Color = clrMuted
									l.MaxLines = 1
									return l.Layout(gtx)
								}),
							)
						})
					})
				}),
			)
			e.Frame(gtx.Ops)
		}
	}
}

// ─── Containers tab ──────────────────────────────────────────────────────────

func (st *stationUI) drawContainersTab(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	const wID, wApp, wPort, wCmd, wAge, wAct float32 = 0.11, 0.17, 0.07, 0.30, 0.16, 0.19
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return withBg(gtx, clrHead, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(wID, headCell(th, "Container ID")),
					layout.Flexed(wApp, headCell(th, "App")),
					layout.Flexed(wPort, headCell(th, "Port")),
					layout.Flexed(wCmd, headCell(th, "Command")),
					layout.Flexed(wAge, headCell(th, "Age")),
					layout.Flexed(wAct, headCell(th, "Actions")),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return divider(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			st.mu.Lock()
			rows := st.containers
			st.mu.Unlock()
			if len(rows) == 0 {
				return emptyMsg(th, "◻", "No containers running",
					"station run [--app <name>] [--port <p>] <dir> <cmd>")(gtx)
			}
			return st.cList.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				row := &rows[i]
				for row.stop.Clicked(gtx) {
					st.runAction(w, "stop", row.data.ID)
				}
				for row.logs.Clicked(gtx) {
					st.openLogsWindow(row.data.ID)
				}
				bg := clrEven
				if i%2 == 1 {
					bg = clrOdd
				}
				d := row.data
				portStr := ""
				if d.Port > 0 {
					portStr = numStr(d.Port)
				}
				cmdStr := strings.Join(d.Command, " ")
				age := time.Since(d.Started).Round(time.Second).String()
				return withBg(gtx, bg, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(wID, cell(th, unit.Sp(11), d.ID, clrMuted)),
						layout.Flexed(wApp, cell(th, unit.Sp(13), d.App, clrDark)),
						layout.Flexed(wPort, cell(th, unit.Sp(13), portStr, clrMuted)),
						layout.Flexed(wCmd, cell(th, unit.Sp(12), cmdStr, clrMuted)),
						layout.Flexed(wAge, cell(th, unit.Sp(12), age, clrMuted)),
						layout.Flexed(wAct, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return btn(gtx, th, &row.stop, "Stop", clrBtnRed)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return btn(gtx, th, &row.logs, "Logs ↗", clrBtnBlue)
									}),
								)
							})
						}),
					)
				})
			})
		}),
	)
}

// ─── Builds tab ──────────────────────────────────────────────────────────────

func (st *stationUI) drawBuildsTab(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	const wID, wApp, wTime, wAct float32 = 0.30, 0.30, 0.25, 0.15
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return withBg(gtx, clrHead, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(wID, headCell(th, "Build ID")),
					layout.Flexed(wApp, headCell(th, "App")),
					layout.Flexed(wTime, headCell(th, "Time")),
					layout.Flexed(wAct, headCell(th, "Actions")),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return divider(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			st.mu.Lock()
			rows := st.builds
			st.mu.Unlock()
			if len(rows) == 0 {
				return emptyMsg(th, "⚙", "No build history",
					"station build [--app <name>] <dir> <cmd>  ·  station build-dockerfile <file> <ctx> <out>")(gtx)
			}
			return st.bList.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				row := &rows[i]
				for row.logs.Clicked(gtx) {
					st.openBuildLogs(row.id)
				}
				bg := clrEven
				if i%2 == 1 {
					bg = clrOdd
				}
				timeStr := ""
				if !row.started.IsZero() {
					timeStr = row.started.Format("2006-01-02 15:04:05")
				}
				return withBg(gtx, bg, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(wID, cell(th, unit.Sp(11), row.id, clrMuted)),
						layout.Flexed(wApp, cell(th, unit.Sp(13), row.app, clrDark)),
						layout.Flexed(wTime, cell(th, unit.Sp(12), timeStr, clrMuted)),
						layout.Flexed(wAct, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return btn(gtx, th, &row.logs, "Logs ↗", clrBtnBlue)
							})
						}),
					)
				})
			})
		}),
	)
}

// ─── Images tab ──────────────────────────────────────────────────────────────

func (st *stationUI) drawImagesTab(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return withBg(gtx, clrHead, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(0.80, headCell(th, "Image")),
					layout.Flexed(0.20, headCell(th, "Actions")),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return divider(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			st.mu.Lock()
			rows := st.images
			st.mu.Unlock()
			if len(rows) == 0 {
				return emptyMsg(th, "□", "No cached images",
					"station image pull <image>  ·  station image list")(gtx)
			}
			return st.iList.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				row := &rows[i]
				for row.rm.Clicked(gtx) {
					st.runAction(w, "image", "rm", row.name)
				}
				bg := clrEven
				if i%2 == 1 {
					bg = clrOdd
				}
				return withBg(gtx, bg, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(0.80, cell(th, unit.Sp(13), row.name, clrDark)),
						layout.Flexed(0.20, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return btn(gtx, th, &row.rm, "Remove", clrBtnRed)
							})
						}),
					)
				})
			})
		}),
	)
}

// ─── Ports tab ───────────────────────────────────────────────────────────────

func (st *stationUI) drawPortsTab(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return withBg(gtx, clrHead, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(0.40, headCell(th, "App")),
					layout.Flexed(0.20, headCell(th, "Port")),
					layout.Flexed(0.20, headCell(th, "Status")),
					layout.Flexed(0.20, headCell(th, "Actions")),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return divider(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			st.mu.Lock()
			rows := st.ports
			st.mu.Unlock()
			if len(rows) == 0 {
				return emptyMsg(th, "⊙", "No port allocations",
					"Ports are auto-allocated when you run containers with --app")(gtx)
			}
			return st.pList.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				row := &rows[i]
				for row.free.Clicked(gtx) {
					st.runAction(w, "port", "free", row.app)
				}
				bg := clrEven
				if i%2 == 1 {
					bg = clrOdd
				}
				status := "idle"
				statusClr := clrMuted
				if !portFree(row.port) {
					status = "● in use"
					statusClr = clrGreen
				}
				return withBg(gtx, bg, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(0.40, cell(th, unit.Sp(13), row.app, clrDark)),
						layout.Flexed(0.20, cell(th, unit.Sp(13), numStr(row.port), clrMuted)),
						layout.Flexed(0.20, cell(th, unit.Sp(12), status, statusClr)),
						layout.Flexed(0.20, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return btn(gtx, th, &row.free, "Free", clrBtnOrange)
							})
						}),
					)
				})
			})
		}),
	)
}

// ─── Snapshots tab ───────────────────────────────────────────────────────────

func (st *stationUI) drawSnapshotsTab(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return withBg(gtx, clrHead, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(0.72, headCell(th, "Snapshot Name")),
					layout.Flexed(0.28, headCell(th, "Actions")),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return divider(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			st.mu.Lock()
			rows := st.snapshots
			st.mu.Unlock()
			if len(rows) == 0 {
				return emptyMsg(th, "◈", "No snapshots saved",
					"station snapshot save <name> <dir>  ·  station snapshot list")(gtx)
			}
			return st.sList.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				row := &rows[i]
				for row.del.Clicked(gtx) {
					st.runAction(w, "snapshot", "delete", row.name)
				}
				for row.load.Clicked(gtx) {
					st.mu.Lock()
					st.openDialog(dialogSnapshotLoad,
						"Load Snapshot: "+row.name,
						"Destination directory", "", "", "", row.name)
					st.mu.Unlock()
				}
				bg := clrEven
				if i%2 == 1 {
					bg = clrOdd
				}
				return withBg(gtx, bg, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(0.72, cell(th, unit.Sp(13), row.name, clrDark)),
						layout.Flexed(0.28, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return btn(gtx, th, &row.load, "Load", clrBtnGreen)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return btn(gtx, th, &row.del, "Delete", clrBtnRed)
									}),
								)
							})
						}),
					)
				})
			})
		}),
	)
}

// ─── Proxies tab ─────────────────────────────────────────────────────────────

func (st *stationUI) drawProxiesTab(gtx layout.Context, th *material.Theme, w *app.Window) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return withBg(gtx, clrHead, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(0.20, headCell(th, "App")),
					layout.Flexed(0.12, headCell(th, "Proxy Port")),
					layout.Flexed(0.30, headCell(th, "Active Upstream")),
					layout.Flexed(0.14, headCell(th, "Slot")),
					layout.Flexed(0.24, headCell(th, "Actions")),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return divider(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			st.mu.Lock()
			rows := st.proxies
			st.mu.Unlock()
			if len(rows) == 0 {
				return emptyMsg(th, "⇄", "No proxies running",
					"station proxy start --app <name> --port <p> --upstream <host:port>")(gtx)
			}
			return st.xList.Layout(gtx, len(rows), func(gtx layout.Context, i int) layout.Dimensions {
				row := &rows[i]
				for row.stop.Clicked(gtx) {
					st.runAction(w, "proxy", "stop", row.data.App)
				}
				for row.swap.Clicked(gtx) {
					st.mu.Lock()
					st.openDialog(dialogProxySwap,
						"Swap upstream: "+row.data.App,
						"New upstream (host:port)", "", row.data.ActiveUpstream, "", row.data.App)
					st.mu.Unlock()
				}
				bg := clrEven
				if i%2 == 1 {
					bg = clrOdd
				}
				d := row.data
				return withBg(gtx, bg, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(0.20, cell(th, unit.Sp(13), d.App, clrDark)),
						layout.Flexed(0.12, cell(th, unit.Sp(13), numStr(d.ProxyPort), clrMuted)),
						layout.Flexed(0.30, cell(th, unit.Sp(12), d.ActiveUpstream, clrMuted)),
						layout.Flexed(0.14, cell(th, unit.Sp(12), d.ActiveSlot, clrMuted)),
						layout.Flexed(0.24, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return btn(gtx, th, &row.swap, "Swap ↔", clrBtnIndigo)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										return btn(gtx, th, &row.stop, "Stop", clrBtnRed)
									}),
								)
							})
						}),
					)
				})
			})
		}),
	)
}
