package ui

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gchaincl/mempool/client"
	"github.com/jroimartin/gocui"
)

const (
	BLOCK_WIDTH       = 22
	BLOCKS_TO_DISPLAY = 4
)

type UI struct {
	gui *gocui.Gui
}

func New() (*UI, error) {
	gui, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		return nil, err
	}

	gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)

	return &UI{gui: gui}, nil
}

func quit(*gocui.Gui, *gocui.View) error { return gocui.ErrQuit }

func (ui *UI) Close() { ui.gui.Close() }

func (ui *UI) Loop() error {
	err := ui.gui.MainLoop()
	// Mask ErrQuit
	if err == gocui.ErrQuit {
		return nil
	}
	return err
}

func (ui *UI) Render(resp *client.Response) {
	ui.gui.Update(func(g *gocui.Gui) error {
		return ui.update(g, resp)
	})
}

func (ui *UI) update(g *gocui.Gui, resp *client.Response) error {
	x, y := g.Size()

	// whether or not use vertical layout
	vertical := BLOCK_WIDTH*5 > x

	// draw projected blocks (mempool)
	for i, _ := range resp.ProjectedBlocks {
		name := fmt.Sprintf("projected-block-%d", i)
		var x0, x1, y0, y1 int
		if vertical {
			x0 = x - (x/4)*(i+1)
			x1 = x0 + BLOCK_WIDTH
			y0 = (y / 2) - 12
			y1 = (y / 2) - 2
		} else {
			x0 = x/2 - BLOCK_WIDTH*(i+1)
			x1 = x0 + BLOCK_WIDTH - 2
			y0 = (y / 2) - 5
			y1 = (y / 2) + 5
		}

		v, err := g.SetView(name, x0, y0, x1, y1)
		if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
		}
		v.Clear()
		if _, err := v.Write(printProjectedBlock(i, resp)); err != nil {
			return err
		}
	}

	if err := ui.separator(g, x, y, vertical); err != nil {
		return err
	}

	// Copy the last BLOCKS_TO_DISPLAY blocks to a slice
	nBlocks := len(resp.Blocks)
	if nBlocks > BLOCKS_TO_DISPLAY {
		nBlocks = BLOCKS_TO_DISPLAY
	}
	blocks := make([]*client.Block, nBlocks)
	for i := 0; i < nBlocks; i++ {
		blocks[i] = &resp.Blocks[len(resp.Blocks)-1-i]
	}

	// draw blockchain blocks
	for i, block := range blocks {
		name := fmt.Sprintf("block-%d", i)
		var x0, x1, y0, y1 int
		if vertical {
			x0 = x - (x/4)*(i+1)
			x1 = x0 + BLOCK_WIDTH
			y0 = (y / 2) + 2
			y1 = (y / 2) + 12
		} else {
			x0 = (x / 2) + (BLOCK_WIDTH*i + 1) + 1
			x1 = x0 + BLOCK_WIDTH - 2
			y0 = (y / 2) - 5
			y1 = (y / 2) + 5
		}

		v, err := g.SetView(name, x0, y0, x1, y1)
		if err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
		}
		v.Title = fmt.Sprintf("#%d", block.Height)
		v.Clear()
		if _, err := v.Write(printBlock(i, blocks)); err != nil {
			return err
		}
	}

	return nil
}

func (ui *UI) separator(g *gocui.Gui, x, y int, vertical bool) error {
	var x0, x1, y0, y1 int
	if vertical {
		x0, x1 = 0, x
		y0, y1 = y/2-1, y/2+1
	} else {
		x0, x1 = x/2-1, x/2+1
		y0, y1 = 0, y
	}

	v, err := g.SetView("separtor", x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Wrap = true
	}

	v.Clear()
	if vertical {
		fmt.Fprintf(v, strings.Repeat("-", x))
	} else {
		fmt.Fprintf(v, strings.Repeat("|", y))
	}

	return nil
}

var (
	white  = color.New(color.FgWhite).SprintfFunc()
	yellow = color.New(color.FgYellow).SprintfFunc()
)

func printProjectedBlock(n int, resp *client.Response) []byte {
	block := resp.ProjectedBlocks[n]

	lines := []string{
		white("   ~%.3f sat/vB       ", block.MedianFee),
		yellow("  %.0f - %.0f sat/vB     ", math.Ceil(block.MinFee), math.Ceil(block.MaxFee)),
		"                       ",
		white("     %.2f MB              ", float64(block.BlockSize)/(1000*1000)),
		white(" %4d transactions     ", block.NTx),
		"                       ",
		"                       ",
		"                       ",
		"                       ",
		"                       ",
	}

	if n < 3 {
		lines[8] = white("   in ~%2d minutes     ", (n+1)*10)
		bg := color.New(color.BgRed).SprintfFunc()
		offset := 9 - int(
			float64(block.BlockWeight)/4000000.0*10,
		)
		for i := offset; i < len(lines); i++ {
			lines[i] = bg("%s", lines[i])
		}
	} else {
		bw := math.Ceil(float64(block.BlockWeight) / 4000000.0)
		lines[8] = white("    +%d blocks", int(bw))
	}

	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, strings.Join(lines, "\n"))
	return buf.Bytes()
}

func printBlock(n int, blocks []*client.Block) []byte {
	block := blocks[n]

	ago := time.Now().Unix() - int64(block.Time)
	lines := []string{
		white("   ~%.3f sat/Vb     ", block.MedianFee),
		yellow("  %.0f - %.0f sat/vB     ", math.Ceil(block.MinFee), math.Ceil(block.MaxFee)),
		"                       ",
		white("     %.2f MB              ", float64(block.Size)/(1000*1000)),
		white(" %4d transactions     ", block.NTx),
		"                       ",
		"                       ",
		"                       ",
		white(" %d secs ago        ", ago),
		"                       ",
	}

	bg := color.New(color.BgBlue).SprintFunc()
	for i, l := range lines {
		lines[i] = bg(l)
	}

	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, strings.Join(lines, "\n"))
	return buf.Bytes()
}