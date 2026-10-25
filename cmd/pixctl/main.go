package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/temamagic/pix-backpack-go/ble"
	"github.com/temamagic/pix-backpack-go/pix"
)

type traceWriter struct{}

func (traceWriter) Write(p []byte) (int, error) {
	fmt.Printf("RPC %s\n", hex.EncodeToString(p))
	return len(p), nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
func run() (err error) {
	if len(os.Args) < 2 {
		return errors.New("usage: pixctl font-license|scan|info|show|stream|live|export|clock|bike|countdown|stopwatch|game|input|text|torch|dice|brightness|demo|clear|probe [flags] [arguments]")
	}
	cmd := os.Args[1]
	valid := map[string]bool{}
	for _, s := range strings.Fields("font-license scan info show stream live export clock bike countdown stopwatch game input text brightness demo clear probe torch dice") {
		valid[s] = true
	}
	if !valid[cmd] {
		return fmt.Errorf("unknown command %q", cmd)
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	match := fs.String("device", "", "exact advertised name or peripheral address")
	timeout := fs.Duration("timeout", 3*time.Second, "scan duration")
	trace := fs.Bool("trace", false, "print outgoing RPC bytes")
	notify := fs.Bool("notify", false, "try subscribing to raw RPC notifications")
	dry := fs.Bool("dry-run", false, "show: print packets without Bluetooth")
	save := fs.Bool("save", false, "show: save animation to flash")
	quantize := fs.Bool("quantize", false, "map colours to the shared web-safe palette")
	index := fs.Int("animation", 0, "zero-based pixpack animation index")
	width := fs.Int("width", 16, "dry-run width")
	height := fs.Int("height", 20, "dry-run height")
	fps := fs.Int("fps", 0, "stream/live: override frame rate (1..60); live defaults to 10")
	duration := fs.Duration("duration", 0, "stream/live: stop after this duration; live defaults to 5s")
	loops := fs.Int("loops", 1, "stream: repeat count, 0 means until cancelled")
	action := fs.String("action", "", "stopwatch: start, stop, reset, lap (without reinitializing)")
	speed := fs.Int("speed", 0, "game: blocks step ms (default 400); text: scroll step ms (default 80)")
	keys := fs.String("keys", "", "game: comma-separated buttons (up, down, left, right, action)")
	interactive := fs.Bool("interactive", false, "game: accept WASD as lines from stdin; q exits")
	gap := fs.Duration("gap", 150*time.Millisecond, "pause between scripted game inputs")
	tz := fs.String("tz", "Europe/Moscow", "clock: IANA timezone")
	fontName := fs.String("font", "mono", "text: mono, mono-bold or demo")
	fontFile := fs.String("font-file", "", "text: custom JSON bitmap font (overrides -font)")
	packPath := fs.String("pack", "", "text: read scrolling text and custom fonts from pixpack")
	textIndex := fs.Int("text-index", 0, "text: scrollingTexts index in pixpack")
	colorHex := fs.String("color", "#ffffff", "text: foreground RGB hex")
	if err := fs.Parse(os.Args[2:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if cmd == "font-license" {
		if fs.NArg() != 0 {
			return errors.New("font-license expects no arguments")
		}
		fmt.Print(pix.BuiltinFontLicense())
		return nil
	}
	if cmd == "export" {
		if fs.NArg() != 2 {
			return errors.New("export requires collection.pixpack OUTPUT_DIR")
		}
		items, err := pix.ExportCollection(fs.Arg(0), fs.Arg(1))
		if err != nil {
			return err
		}
		for _, item := range items {
			fmt.Printf("%2d  %-20s %3d frames %3d colours\n", item.Index, item.Name, item.Frames, item.Colors)
		}
		fmt.Printf("exported %d animations to %s/index.html\n", len(items), fs.Arg(1))
		return nil
	}
	want := 0
	switch cmd {
	case "show", "stream", "bike", "countdown", "game", "input", "text", "brightness", "demo", "dice":
		want = 1
	}
	if cmd == "text" && *packPath != "" {
		if *fontFile != "" {
			return errors.New("-font-file and -pack cannot be combined")
		}
		want = 0
	}
	if fs.NArg() != want {
		return fmt.Errorf("%s expects %d arguments; flags must precede them", cmd, want)
	}
	if *dry && cmd != "show" {
		return errors.New("dry-run is supported for show only")
	}
	if *fps < 0 || *fps > 60 || *loops < 0 || *duration < 0 || *gap < 0 {
		return errors.New("invalid fps, loops, duration or gap")
	}
	if *dry {
		a, err := pix.Load(fs.Arg(0), *width, *height, *index)
		if err != nil {
			return err
		}
		if *quantize {
			a = pix.Quantize(a)
		}
		return pix.NewClient(traceWriter{}).Upload(a, 255, *save)
	}
	opts := ble.Options{Device: *match, Timeout: *timeout, Log: os.Stderr}
	if *trace {
		opts.Trace = os.Stdout
	}
	if cmd == "scan" {
		_, err := ble.Scan(opts)
		return err
	}
	connection, err := ble.Open(opts)
	if err != nil {
		return err
	}
	defer func() {
		if e := connection.Close(); err == nil {
			err = e
		}
	}()
	c := connection.Client
	info := connection.Info
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *notify {
		if e := connection.SubscribeRPC(func(p []byte) { fmt.Fprintf(os.Stderr, "NOTIFY %x\n", p) }); e != nil {
			fmt.Fprintln(os.Stderr, "RPC notifications unavailable:", e)
		}
	}
	switch cmd {
	case "probe":
		return probe(ctx, c, info)
	case "info":
		b, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(b))
		return nil
	case "clear":
		return c.Clear()
	case "torch", "dice":
		color, e := pix.ParseRGB(*colorHex)
		if e != nil {
			return e
		}
		if cmd == "torch" {
			return c.Torch(info.Width, info.Height, info.MaxFrames, color)
		}
		n, e := strconv.Atoi(fs.Arg(0))
		if e != nil {
			return e
		}
		return c.Dice(n, info.Width, info.Height, info.MaxFrames, color)
	case "brightness":
		n, e := strconv.Atoi(fs.Arg(0))
		if e != nil {
			return e
		}
		return connection.SetBrightness(n)
	case "demo":
		switch fs.Arg(0) {
		case "on":
			return connection.SetDemo(true)
		case "off":
			return connection.SetDemo(false)
		}
		return errors.New("demo expects on or off")
	case "clock":
		loc, e := time.LoadLocation(*tz)
		if e != nil {
			return e
		}
		o := pix.DefaultClock(time.Now().In(loc))
		if info.Height == 16 {
			o.Position = 4
		}
		return c.Clock(o)
	case "bike":
		stages := map[string]pix.BikeStage{"init": pix.BikeInit, "idle": pix.BikeIdle, "stop": pix.BikeStop, "left": pix.BikeLeft, "right": pix.BikeRight}
		stage, ok := stages[fs.Arg(0)]
		if !ok {
			return errors.New("bike expects init, idle, stop, left or right")
		}
		return c.Bike(stage)
	case "countdown":
		n, e := strconv.Atoi(fs.Arg(0))
		if e != nil {
			return e
		}
		return c.Countdown(n)
	case "stopwatch":
		if *action == "" {
			return c.Stopwatch()
		}
		actions := map[string]pix.StopwatchAction{"start": pix.StopwatchStart, "stop": pix.StopwatchStop, "reset": pix.StopwatchReset, "lap": pix.StopwatchLap}
		v, ok := actions[*action]
		if !ok {
			return errors.New("invalid stopwatch action")
		}
		return c.ControlStopwatch(v)
	case "text":
		if *packPath != "" {
			pack, e := pix.OpenPack(*packPath)
			if e != nil {
				return e
			}
			return pack.Scroll(c, *textIndex, info.Width, info.Height)
		}
		font, e := pix.BuiltinFont(*fontName)
		if *fontFile != "" {
			font, e = pix.LoadFont(*fontFile)
		}
		if e != nil {
			return e
		}
		data, e := hex.DecodeString(strings.TrimPrefix(*colorHex, "#"))
		if e != nil || len(data) != 3 {
			return errors.New("colour must be #RRGGBB")
		}
		rate := *speed
		if rate == 0 {
			rate = 80
		}
		return c.ScrollText(fs.Arg(0), font, pix.RGB{R: data[0], G: data[1], B: data[2]}, rate, info.Width, info.Height)
	case "input":
		button, e := parseButton(fs.Arg(0))
		if e != nil {
			return e
		}
		return c.GameInput(button)
	case "game":
		games := map[string]pix.Game{"crawler": pix.GameCrawler, "blocks": pix.GamePixBlocks, "breaker": pix.GamePixelBreaker}
		game, ok := games[fs.Arg(0)]
		if !ok {
			return errors.New("game expects crawler, blocks or breaker")
		}
		var buttons []pix.Button
		if *keys != "" {
			for _, key := range strings.Split(*keys, ",") {
				b, e := parseButton(key)
				if e != nil {
					return e
				}
				buttons = append(buttons, b)
			}
		}
		gameSpeed := *speed
		if gameSpeed == 0 {
			gameSpeed = 400
		}
		if err = c.StartGame(game, gameSpeed); err != nil {
			return err
		}
		for _, button := range buttons {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(*gap):
			}
			if err = c.GameInput(button); err != nil {
				return err
			}
		}
		if *interactive {
			fmt.Println("Enter w/up, s/down, a/left, d/right, space/action; q exits.")
			lines := make(chan string)
			scanErrors := make(chan error, 1)
			go func() {
				scanner := bufio.NewScanner(os.Stdin)
				defer func() { scanErrors <- scanner.Err(); close(lines) }()
				for scanner.Scan() {
					select {
					case lines <- scanner.Text():
					case <-ctx.Done():
						return
					}
				}
			}()
			for {
				var line string
				select {
				case <-ctx.Done():
					return c.Clear()
				case value, ok := <-lines:
					if !ok {
						return <-scanErrors
					}
					line = strings.TrimSpace(value)
				}
				if line == "q" {
					return c.Clear()
				}
				button, e := parseButton(line)
				if e != nil {
					fmt.Println(e)
					continue
				}
				if e = c.GameInput(button); e != nil {
					return e
				}
			}
		}
		return nil
	case "live":
		rate := *fps
		if rate == 0 {
			rate = 10
		}
		limit := *duration
		if limit == 0 {
			limit = 5 * time.Second
		}
		liveCtx, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		a := pix.Animation{Width: info.Width, Height: info.Height}
		for i := 0; i < info.Width*info.Height; i++ {
			pixels := make([]pix.RGB, info.Width*info.Height)
			pixels[i] = pix.RGB{R: 255, G: 100}
			a.Frames = append(a.Frames, pix.Frame{Pixels: pixels, Duration: max(1, 1000/rate)})
		}
		stats, e := c.Stream(liveCtx, a, 0)
		printStats(stats)
		if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) {
			return nil
		}
		return e
	case "show", "stream":
		a, e := pix.Load(fs.Arg(0), info.Width, info.Height, *index)
		if e != nil {
			return e
		}
		if *quantize {
			a = pix.Quantize(a)
		}
		if cmd == "show" {
			if e = c.Upload(a, info.MaxFrames, *save); e != nil {
				return e
			}
			fmt.Printf("uploaded %d frame(s), %dx%d\n", len(a.Frames), a.Width, a.Height)
			return nil
		}
		if *fps > 0 {
			for i := range a.Frames {
				a.Frames[i].Duration = max(1, 1000 / *fps)
			}
		}
		if *duration > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, *duration)
			defer cancel()
		}
		stats, e := c.Stream(ctx, a, *loops)
		printStats(stats)
		if errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) {
			return nil
		}
		return e
	}
	return errors.New("command not implemented")
}

// probe is an explicit, bounded interoperability experiment, without flash/OTA writes.
func probe(ctx context.Context, c *pix.Client, info ble.Info) error {
	fmt.Printf("device=%s firmware=%s hardware=%s max_frames_raw=%s\n", info.Name, info.Firmware, info.Hardware, info.MaxFramesRaw)
	type step struct {
		name  string
		run   func() error
		pause time.Duration
	}
	steps := []step{
		{"countdown-5s", func() error { return c.Countdown(5) }, 1500 * time.Millisecond},
		{"stopwatch-init", c.Stopwatch, 0}, {"stopwatch-start", func() error { return c.ControlStopwatch(pix.StopwatchStart) }, time.Second},
		{"stopwatch-lap", func() error { return c.ControlStopwatch(pix.StopwatchLap) }, 500 * time.Millisecond}, {"stopwatch-stop", func() error { return c.ControlStopwatch(pix.StopwatchStop) }, 500 * time.Millisecond},
		{"bike-init", func() error { return c.Bike(pix.BikeInit) }, 0}, {"bike-idle", func() error { return c.Bike(pix.BikeIdle) }, 500 * time.Millisecond},
		{"bike-left", func() error { return c.Bike(pix.BikeLeft) }, time.Second}, {"bike-right", func() error { return c.Bike(pix.BikeRight) }, time.Second}, {"bike-stop", func() error { return c.Bike(pix.BikeStop) }, time.Second},
		{"crawler-init", func() error { return c.StartGame(pix.GameCrawler, 400) }, 500 * time.Millisecond},
		{"crawler-right", func() error { return c.GameInput(pix.ButtonRight) }, 500 * time.Millisecond}, {"crawler-down", func() error { return c.GameInput(pix.ButtonDown) }, 500 * time.Millisecond},
		{"blocks-init", func() error { return c.StartGame(pix.GamePixBlocks, 400) }, 500 * time.Millisecond},
		{"blocks-left", func() error { return c.GameInput(pix.ButtonLeft) }, 500 * time.Millisecond}, {"blocks-action", func() error { return c.GameInput(pix.ButtonA) }, 500 * time.Millisecond},
		{"breaker-init", func() error { return c.StartGame(pix.GamePixelBreaker, 400) }, 500 * time.Millisecond}, {"breaker-right", func() error { return c.GameInput(pix.ButtonRight) }, 500 * time.Millisecond},
	}
	for _, s := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		start := time.Now()
		if err := s.run(); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		fmt.Printf("GATT_OK %-20s %s\n", s.name, time.Since(start).Round(time.Millisecond))
		timer := time.NewTimer(s.pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	font, err := pix.BuiltinFont("mono")
	if err != nil {
		return err
	}
	if err = c.ScrollText("PIX from Go", font, pix.RGB{R: 40, G: 220, B: 255}, 80, info.Width, info.Height); err != nil {
		return err
	}
	fmt.Println("GATT_OK scrolling-text (left active)")
	return nil
}
func parseButton(s string) (pix.Button, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "w", "up", "1":
		return pix.ButtonUp, nil
	case "s", "down", "2":
		return pix.ButtonDown, nil
	case "a", "left", "3":
		return pix.ButtonLeft, nil
	case "d", "right", "4":
		return pix.ButtonRight, nil
	case "space", "action", "5":
		return pix.ButtonA, nil
	}
	return 0, fmt.Errorf("unknown button %q", s)
}
func printStats(s pix.StreamStats) {
	rate := 0.0
	if s.Elapsed > 0 {
		rate = float64(s.Frames) / s.Elapsed.Seconds()
	}
	fmt.Printf("frames=%d rpc_bytes=%d elapsed=%s actual_fps=%.2f (setup excluded)\n", s.Frames, s.Bytes, s.Elapsed.Round(time.Millisecond), rate)
}
