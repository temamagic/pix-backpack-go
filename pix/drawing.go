package pix

import "fmt"

// Torch displays a solid colour using the normal animation protocol.
func (c *Client) Torch(width, height, maxFrames int, color RGB) error {
	if width < 1 || height < 1 || width > 255 || height > 255 || width*height > 506 {
		return fmt.Errorf("invalid dimensions")
	}
	pixels := make([]RGB, width*height)
	for i := range pixels {
		pixels[i] = color
	}
	return c.Upload(Animation{Width: width, Height: height, Frames: []Frame{{Pixels: pixels, Duration: 300}}}, maxFrames, false)
}

// Dice renders a supplied die value. Randomness is a host concern; there is no
// separate firmware DICE opcode in the recovered protocol.
func (c *Client) Dice(value, width, height, maxFrames int, color RGB) error {
	if value < 1 || value > 6 || width < 10 || height < 10 || width > 255 || height > 255 || width*height > 506 {
		return fmt.Errorf("invalid die value or dimensions")
	}
	pixels := make([]RGB, width*height)
	positions := [][2]int{}
	if value%2 == 1 {
		positions = append(positions, [2]int{width / 2, height / 2})
	}
	if value >= 2 {
		positions = append(positions, [2]int{width / 4, height / 4}, [2]int{width * 3 / 4, height * 3 / 4})
	}
	if value >= 4 {
		positions = append(positions, [2]int{width * 3 / 4, height / 4}, [2]int{width / 4, height * 3 / 4})
	}
	if value == 6 {
		positions = append(positions, [2]int{width / 4, height / 2}, [2]int{width * 3 / 4, height / 2})
	}
	for _, p := range positions {
		for y := p[1] - 1; y <= p[1]; y++ {
			for x := p[0] - 1; x <= p[0]; x++ {
				pixels[y*width+x] = color
			}
		}
	}
	return c.Upload(Animation{Width: width, Height: height, Frames: []Frame{{Pixels: pixels, Duration: 300}}}, maxFrames, false)
}
