package SSD1306go

import (
	"errors"
	"image"
	"time"

	"github.com/stianeikeland/go-rpio"
	"golang.org/x/exp/io/i2c"
)

var (
	// ErroledContrast コントラスト値が適切ではなかったとき
	ErroledContrast error = errors.New("コントラストの値は1~256です")
)

// SSD1306 SSD1306
type SSD1306 struct {
	Width  int
	Height int
	Name   string
	Addr   int
	i2c    *i2c.Device
	buffer []byte
}

// NewSSD1306 SSD1306
func NewSSD1306(width, height int, name string, addr int) (*SSD1306, error) {
	ssd := &SSD1306{
		Width:  width,
		Height: height,
		Name:   name,
		Addr:   addr,
	}

	d, err := i2c.Open(&i2c.Devfs{Dev: ssd.Name}, ssd.Addr)
	if err != nil {
		return nil, err
	}

	ssd.i2c = d
	ssd.buffer = make([]byte, 128*4)
	return ssd, nil
}

// Init 初期設定
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

// Halt OLEDをストップします
func (oled *SSD1306) Halt() {
	oled.Clear()
	oled.Display()
	oled.reset()
}

func (oled *SSD1306) reset() {
	rpio.Open()
	defer rpio.Close()

	pin := rpio.Pin(4)
	pin.Output()

	pin.High()
	time.Sleep(time.Millisecond)
	pin.Low()
	time.Sleep(10 * time.Millisecond)
	pin.High()
}

// SetPixel 指定したい位置にドットを書きます
func (oled *SSD1306) SetPixel(x, y int, inverse bool) {
	if inverse {
		oled.buffer[x+(y/8)*oled.Width] ^= (1 << (y & 7))
	} else {
		oled.buffer[x+(y/8)*oled.Width] |= (1 << (y & 7))
	}
}

// Buffer 生のバッファーです
func (oled *SSD1306) Buffer() *[]byte {
	return &oled.buffer
}

// Display ディスプレイにバッファーを書き込みます
func (oled *SSD1306) Display() error {
	return oled.cmds(append([]byte{0x40}, oled.buffer...))
}

// DisplayOn ディスプレイをオンにします
func (oled *SSD1306) DisplayOn() {
	oled.cmd(byte(0xAF))
}

// DisplayOff ディスプレイをオフにします
func (oled *SSD1306) DisplayOff() {
	oled.cmd(byte(0xAE))
}

// DisplayInvert ディスプレイの背景色を反転させます
func (oled *SSD1306) DisplayInvert(invert bool) {
	if invert {
		oled.cmd(byte(0xA7))
	} else {
		oled.cmd(byte(0xA6))
	}
}

// SetRotation ディスプレイの表示を回転します
func (oled *SSD1306) SetRotation(n uint8) {
	// 垂直 0xC0 / 0xC8
	// 水平 0xA0 / 0xA1

	// oled.cmd(byte())
}

// SetContrast コントラストを調整します (デフォルトは0x7F)
func (oled *SSD1306) SetContrast(contrast int) error {
	if contrast <= 1 || contrast >= 256 {
		return ErroledContrast
	}

	oled.cmd(byte(0x81))
	oled.cmd(byte(contrast))

	return nil
}

// Blink ディスプレイを点滅させます
func (oled *SSD1306) Blink() {
	oled.DisplayOff()
	time.Sleep(80 * time.Millisecond)
	oled.DisplayOn()
	time.Sleep(80 * time.Millisecond)
	oled.DisplayOff()
	time.Sleep(80 * time.Millisecond)
	oled.DisplayOn()
}

// Clear バッファーをクリアします
func (oled *SSD1306) Clear() {
	oled.buffer = make([]byte, 512)
	// oled.Display()
}

// SetImage バッファーへ画像をセットします
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

// SetImageRGBA image.RGBAの画像をバッファーへセットします
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

func (oled *SSD1306) cmd(d byte) error {
	return oled.i2c.Write([]byte{0x80, d})
}

func (oled *SSD1306) cmds(commands []byte) error {
	return oled.i2c.Write(commands)
}
