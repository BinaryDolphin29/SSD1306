package SSD1306

import (
	"image"
	"log"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
	"periph.io/x/host/v3/rpi"
)

// SSD1306 SSD1306
type SSD1306 struct {
	Width  int
	Height int
	Name   string
	Addr   int
	com    struct {
		i2c    *i2c.Dev
		closer i2c.BusCloser
	}

	buffer []byte
}

// NewSSD1306 SSD1306
func NewSSD1306(width, height int, name string, addr uint16) (*SSD1306, error) {
	oled := &SSD1306{
		Width:  width,
		Height: height,
		Name:   name,
		Addr:   int(addr),
	}

	if _, err := host.Init(); err != nil {
		log.Fatalln(err.Error())
	}

	b, err := i2creg.Open("/dev/i2c-1")
	if err != nil {
		log.Fatal(err)
	}

	oled.com.i2c = &i2c.Dev{Addr: addr, Bus: b}
	oled.com.closer = b
	oled.buffer = make([]byte, (width*height)/8)

	return oled, nil
}

// Init Initializing process.
func (oled *SSD1306) Init() {
	oled.reset()
	oled.cmds([]byte{
		0x00,
		0xAE,       // 画面を消す
		0x20, 0x00, // メモリモード
		0xD5, 0x80, // ディスプレイクロック
		0xA8, byte(oled.Height - 1), // 高さ
		0xD3, 0x00, // 表示オフセット
		0x40 | 0x00, // 開始ライン
		0x8D, 0x14,  // チャージポンプ
		0xA0 | 0x1, // 水平
		0xC8,       // 垂直
		0xDA, 0x02, // COM ピン
		0x81, 0x7F, // コントラスト
		0xD9, 0xF1, // pre charge
		0xDB, 0x40, // vcomh
		0x21, 0x00, byte(oled.Width - 1), // カラムのスタートアドレス
		0x22, 0x00, byte(oled.Height/8 - 1), // ページのスタートアドレス
		0xA4, // GDDRAMを使う
		0xA6, // 普通のディスプレイ
		0xAF, // 画面を表示
	})
}

// Close Run Clear then Display and close i2c driver.
func (oled *SSD1306) Close() {
	oled.Clear()
	oled.Display()

	oled.com.closer.Close()
}

func (oled *SSD1306) reset() {
	rstPin := rpi.P1_7

	rstPin.Out(gpio.High)
	time.Sleep(time.Millisecond)
	rstPin.Out(gpio.Low)
	time.Sleep(10 * time.Millisecond)
	rstPin.Out(gpio.High)
}

// SetPixel Set the pixel at the buffer.
func (oled *SSD1306) SetPixel(x, y int, inverse bool) {
	if inverse {
		oled.buffer[x+(y/8)*oled.Width] ^= (1 << (y & 7))
	} else {
		oled.buffer[x+(y/8)*oled.Width] |= (1 << (y & 7))
	}
}

// Buffer Raw buffer.
func (oled *SSD1306) Buffer() *[]byte {
	return &oled.buffer
}

// Display Write the buffer at the display.
func (oled *SSD1306) Display() (int, error) {
	return oled.cmds(append([]byte{0x40}, oled.buffer...))
}

// DisplayOn TurnOn the display.
func (oled *SSD1306) DisplayOn() {
	oled.cmd(byte(0xAF))
}

// DisplayOff TurnOff the display.
func (oled *SSD1306) DisplayOff() {
	oled.cmd(byte(0xAE))
}

// DisplayInvert Invert the display pixel.
func (oled *SSD1306) DisplayInvert(invert bool) {
	if invert {
		oled.cmd(0xA7)
	} else {
		oled.cmd(0xA6)
	}
}

// SetRotation Unimplemented
func (oled *SSD1306) SetRotation(n uint8) {
	// 垂直 0xC0 / 0xC8
	// 水平 0xA0 / 0xA1

	// oled.cmd(byte())
}

// SetContrast Adjustment the display contrast. default is 0x7F.
func (oled *SSD1306) SetContrast(contrast uint8) {
	oled.cmd(0x81)
	oled.cmd(contrast + 1)
}

// Blink Blink the display ones.
func (oled *SSD1306) Blink(t time.Duration) {
	oled.DisplayOff()
	time.Sleep(t)
	oled.DisplayOn()
}

// Clear Clear the buffer. warn: does not run Display().
func (oled *SSD1306) Clear() {
	oled.buffer = make([]byte, (oled.Width*oled.Height)/8)
}

// SetImage Set the image (image.Image) at the buffer.
func (oled *SSD1306) SetImage(img image.Image) error {
	bounds := img.Bounds()
	// if bounds.Max.X > oled.Height || bounds.Max.Y > oled.Width {
	// 	return errors.New("画像の解像度がディスプレイの解像度を上回っています。")
	// }
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			ans := r + g + b
			if ans > 0 {
				oled.SetPixel(x, y, false)
			}
		}
	}

	return nil
}

// SetImageRGBA Set the image (image.RGBA) at the buffer.
func (oled *SSD1306) SetImageRGBA(img image.RGBA) error {
	bounds := img.Bounds()
	// if bounds.Max.X > oled.Height || bounds.Max.Y > oled.Width {
	// 	return errors.New("画像の解像度がディスプレイの解像度を上回っています。")
	// }
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				oled.SetPixel(x, y, false)
			}
		}
	}
	return nil
}

func (oled *SSD1306) cmd(d byte) (int, error) {
	return oled.com.i2c.Write([]byte{0x80, d})
}

func (oled *SSD1306) cmds(commands []byte) (int, error) {
	return oled.com.i2c.Write(commands)
}
