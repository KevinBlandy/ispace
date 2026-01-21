package util

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand/v2"
)

type AvatarGenerator struct {
	Size      int
	GridCount int
}

// NewAvatarGenerator 创建 Hash 头像生成器
func NewAvatarGenerator(size, gridCount int) *AvatarGenerator {
	return &AvatarGenerator{
		Size:      size,      // 大小，像素
		GridCount: gridCount, // 行列数量
	}
}

// GenerateToBytes 生成每个色块颜色都随机的头像
func (g *AvatarGenerator) GenerateToBytes(id int64) ([]byte, error) {
	// 1. 初始化随机数生成器 (v2)
	s1, s2 := uint64(id), uint64(id)^0xDEADC0DE
	r := rand.New(rand.NewPCG(s1, s2))

	// 2. 随机背景色 (固定在较浅范围)
	bgColor := color.RGBA{
		R: uint8(r.UintN(30) + 225), // 225-255
		G: uint8(r.UintN(30) + 225),
		B: uint8(r.UintN(30) + 225),
		A: 255,
	}

	// 3. 创建画布并填充背景
	img := image.NewRGBA(image.Rect(0, 0, g.Size, g.Size))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	// 4. 绘制网格
	cellSize := g.Size / g.GridCount
	padding := 1 // 每个色块边缘缩进1px，两个色块间即形成2px间隔

	for x := 0; x < g.GridCount; x++ {
		for y := 0; y < g.GridCount; y++ {
			// 每个色块 2/3 的概率出块，增加画面丰富度
			if r.UintN(3) > 0 {
				// 为当前色块生成独立的随机颜色
				blockColor := color.RGBA{
					R: uint8(r.UintN(200)), // 颜色范围较广
					G: uint8(r.UintN(200)),
					B: uint8(r.UintN(200)),
					A: 255,
				}

				// 计算坐标并应用 1px 的缩进
				rect := image.Rect(
					x*cellSize+padding,
					y*cellSize+padding,
					(x+1)*cellSize-padding,
					(y+1)*cellSize-padding,
				)

				draw.Draw(img, rect, &image.Uniform{C: blockColor}, image.Point{}, draw.Src)
			}
		}
	}

	// 5. 导出为 PNG []byte
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
