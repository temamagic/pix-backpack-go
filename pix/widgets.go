package pix

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// BikeStage selects the firmware bike indicator state.
type BikeStage byte

// Bike indicator stages.
const (
	BikeInit BikeStage = iota
	BikeIdle
	BikeStop
	BikeLeft
	BikeRight
)

// Bike sends the selected bike indicator stage.
func (c *Client) Bike(stage BikeStage) error {
	if stage > BikeRight {
		return fmt.Errorf("invalid bike stage")
	}
	return c.SetConfig(0, []byte{byte(ModeBike), byte(stage), 0})
}

// Countdown sets a firmware countdown between 0 and 5999 seconds.
func (c *Client) Countdown(seconds int) error {
	if seconds < 0 || seconds > 5999 {
		return fmt.Errorf("countdown must be 0..5999 seconds")
	}
	p := binary.LittleEndian.AppendUint16([]byte{byte(ModeCountdown), 0}, uint16(seconds))
	return c.SetConfig(0, append(p, 5))
}

// Stopwatch matches iOS encodeStopwatchConfig, including eight ff sentinel bytes.
func (c *Client) Stopwatch() error {
	return c.SetConfig(0, []byte{9, 0, 1, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255})
}

// StopwatchAction selects a stopwatch input event.
type StopwatchAction byte

// Stopwatch input events.
const (
	StopwatchStart StopwatchAction = 1
	StopwatchStop  StopwatchAction = 2
	StopwatchReset StopwatchAction = 3
	StopwatchLap   StopwatchAction = 4
)

// ControlStopwatch sends a stopwatch action without reinitializing its configuration.
func (c *Client) ControlStopwatch(action StopwatchAction) error {
	if action < 1 || action > 4 {
		return fmt.Errorf("invalid stopwatch action")
	}
	return c.Input(byte(action))
}

// Game identifies a built-in firmware game.
type Game Mode

// Firmware-resident game modes.
const (
	GameCrawler      Game = Game(ModeCrawler)
	GamePixBlocks    Game = Game(ModePixBlocks)
	GamePixelBreaker Game = Game(ModePixelBreaker)
)

// Button identifies a firmware game input.
type Button byte

// Game input button codes.
const (
	ButtonUp    Button = 1
	ButtonDown  Button = 2
	ButtonLeft  Button = 3
	ButtonRight Button = 4
	ButtonA     Button = 5
)

// StartGame runs a firmware game. No game executable or rendered frames are sent.
func (c *Client) StartGame(game Game, speedMS int) error {
	switch game {
	case GameCrawler, GamePixelBreaker:
		return c.SetConfig(0, []byte{byte(game), 0})
	case GamePixBlocks:
		if speedMS < 1 || speedMS > 65535 {
			return fmt.Errorf("blocks speed must be 1..65535 ms")
		}
		p := binary.LittleEndian.AppendUint16([]byte{byte(game)}, uint16(speedMS))
		return c.SetConfig(0, append(p, 0))
	}
	return fmt.Errorf("unknown game %d", game)
}

// GameInput sends one directional or action button event.
func (c *Client) GameInput(button Button) error {
	if button < 1 || button > 5 {
		return fmt.Errorf("invalid button")
	}
	return c.Input(byte(button))
}

// ClockOptions configures local wall-clock time, placement, format and colors.
type ClockOptions struct {
	Time                   time.Time
	Format12               bool
	Position               uint16 // 6 for original, 4 for Mini
	Numbers, Marker, Alarm RGB
}

// DefaultClock returns white numbers, red accents and the Original display position.
func DefaultClock(t time.Time) ClockOptions {
	return ClockOptions{Time: t, Position: 6, Numbers: RGB{255, 255, 255}, Marker: RGB{255, 0, 0}, Alarm: RGB{255, 0, 0}}
}

// Clock sends iOS's local wall-clock epoch and IANA timezone string. No alarm
// is armed (alarm time is in the past, alarm duration=0).
func (c *Client) Clock(o ClockOptions) error {
	if o.Time.IsZero() {
		return fmt.Errorf("clock time is required")
	}
	_, offset := o.Time.Zone()
	stamp := o.Time.Unix() + int64(offset)
	if stamp < 60 || stamp > int64(^uint32(0)) {
		return fmt.Errorf("clock timestamp outside uint32 range")
	}
	tz := o.Time.Location().String()
	if strings.IndexByte(tz, 0) >= 0 || len(tz) > 120 {
		return fmt.Errorf("invalid timezone")
	}
	p := binary.LittleEndian.AppendUint32([]byte{6}, uint32(stamp))
	p = binary.LittleEndian.AppendUint32(p, uint32(stamp-60))
	format := byte(1)
	if o.Format12 {
		format = 0
	}
	p = append(p, 0, 0, 1, format)
	p = binary.LittleEndian.AppendUint16(p, o.Position)
	p = append(p, 0, 0, 0, 0, 1, 1, 2)
	p = append(p, []byte(tz)...)
	p = append(p, 0, 0)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.setPalette(0, []RGB{o.Numbers, o.Marker, o.Alarm}); err != nil {
		return err
	}
	return c.setConfig(0, p)
}
