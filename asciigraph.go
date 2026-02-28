package asciigraph

import (
	"fmt"
	"math"

	"github.com/pancsta/tcell-v2"
)

// PlotToScreen renders an ascii graph for a series directly to a tcell.Screen.
// The graph is rendered starting at the specified x, y coordinates.
func PlotToScreen(screen tcell.Screen, x, y int, series []float64, options ...Option) {
	PlotManyToScreen(screen, x, y, [][]float64{series}, options...)
}

// PlotManyToScreen renders an ascii graph for multiple series directly to a tcell.Screen.
// The graph is rendered starting at the specified x, y coordinates.
func PlotManyToScreen(screen tcell.Screen, x, y int, data [][]float64, options ...Option) {
	var logMaximum float64
	config := configure(config{
		Offset:    3,
		Precision: 2,
	}, options)

	if config.HideAxisY {
		x -= 3
		if config.Width > 0 {
			config.Width += 3
		}
	}
	if config.HideAxisX {
		config.Height++
	}

	// Create a deep copy of the input data
	dataCopy := make([][]float64, len(data))
	for i, series := range data {
		dataCopy[i] = make([]float64, len(series))
		copy(dataCopy[i], series)
	}
	data = dataCopy

	lenMax := 0
	for i := range data {
		if l := len(data[i]); l > lenMax {
			lenMax = l
		}
	}

	if config.Width > 0 {
		for i := range data {
			for j := len(data[i]); j < lenMax; j++ {
				data[i] = append(data[i], math.NaN())
			}
			data[i] = interpolateArray(data[i], config.Width)
		}

		lenMax = config.Width
	}

	minimum, maximum := math.Inf(1), math.Inf(-1)
	for i := range data {
		minVal, maxVal := minMaxFloat64Slice(data[i])
		if minVal < minimum {
			minimum = minVal
		}
		if maxVal > maximum {
			maximum = maxVal
		}
	}
	if config.LowerBound != nil && *config.LowerBound < minimum {
		minimum = *config.LowerBound
	}
	if config.UpperBound != nil && *config.UpperBound > maximum {
		maximum = *config.UpperBound
	}
	interval := math.Abs(maximum - minimum)

	if config.Height <= 0 {
		config.Height = calculateHeight(interval)
	}

	if config.Offset <= 0 {
		config.Offset = 3
	}

	var ratio float64
	if interval != 0 {
		ratio = float64(config.Height) / interval
	} else {
		ratio = 1
	}
	min2 := round(minimum * ratio)
	max2 := round(maximum * ratio)

	intmin2 := int(min2)
	intmax2 := int(max2)

	rows := int(math.Abs(float64(intmax2 - intmin2)))
	width := lenMax + config.Offset

	type cell struct {
		Text  string
		Color tcell.Color
	}
	plot := make([][]cell, rows+1)

	// initialise empty 2D grid
	for i := 0; i < rows+1; i++ {
		line := make([]cell, width)
		for j := 0; j < width; j++ {
			line[j].Text = " "
			line[j].Color = tcell.ColorDefault
		}
		plot[i] = line
	}

	precision := config.Precision
	logMaximum = math.Log10(math.Max(math.Abs(maximum), math.Abs(minimum))) // to find number of zeros after decimal
	if minimum == float64(0) && maximum == float64(0) {
		logMaximum = float64(-1)
	}

	if logMaximum < 0 {
		// negative log
		if math.Mod(logMaximum, 1) != 0 {
			// non-zero digits after decimal
			precision += uint(math.Abs(logMaximum))
		} else {
			precision += uint(math.Abs(logMaximum) - 1.0)
		}
	} else if logMaximum > 2 {
		precision = 0
	}

	maxNumLength := len(fmt.Sprintf("%0.*f", precision, maximum))
	minNumLength := len(fmt.Sprintf("%0.*f", precision, minimum))
	maxWidth := int(math.Max(float64(maxNumLength), float64(minNumLength)))

	// axis and labels
	if !config.HideAxisY {
		for yAxis := intmin2; yAxis < intmax2+1; yAxis++ {
			var magnitude float64
			if rows > 0 {
				magnitude = maximum - (float64(yAxis-intmin2) * interval / float64(rows))
			} else {
				magnitude = float64(yAxis)
			}

			label := fmt.Sprintf("%*.*f", maxWidth+1, precision, magnitude)
			w := yAxis - intmin2
			h := int(math.Max(float64(config.Offset)-float64(len(label)), 0))

			plot[w][h].Text = label
			plot[w][h].Color = config.LabelColor
			plot[w][config.Offset-1].Text = "┤"
			plot[w][config.Offset-1].Color = config.AxisColor
		}
	}

	for i := range data {
		series := data[i]

		color := tcell.ColorDefault
		if i < len(config.SeriesColors) {
			color = config.SeriesColors[i]
		}

		var y0, y1 int

		if !math.IsNaN(series[0]) && !config.HideAxisY {
			y0 = int(round(series[0]*ratio) - min2)
			plot[rows-y0][config.Offset-1].Text = "┼" // first value
			plot[rows-y0][config.Offset-1].Color = config.AxisColor
		}

		for xAxis := 0; xAxis < len(series)-1; xAxis++ { // plot the line
			d0 := series[xAxis]
			d1 := series[xAxis+1]

			// if d1 == 0 && d0 == 0 {
			// 	continue
			// }

			if math.IsNaN(d0) && math.IsNaN(d1) {
				continue
			}

			if math.IsNaN(d1) && !math.IsNaN(d0) {
				y0 = int(round(d0*ratio) - float64(intmin2))
				plot[rows-y0][xAxis+config.Offset].Text = "╴"
				plot[rows-y0][xAxis+config.Offset].Color = color
				continue
			}

			if math.IsNaN(d0) && !math.IsNaN(d1) {
				y1 = int(round(d1*ratio) - float64(intmin2))
				plot[rows-y1][xAxis+config.Offset].Text = "╶"
				plot[rows-y1][xAxis+config.Offset].Color = color
				continue
			}

			y0 = int(round(d0*ratio) - float64(intmin2))
			y1 = int(round(d1*ratio) - float64(intmin2))

			if y0 == y1 {
				plot[rows-y0][xAxis+config.Offset].Text = "─"
			} else {
				if y0 > y1 {
					plot[rows-y1][xAxis+config.Offset].Text = "╰"
					plot[rows-y0][xAxis+config.Offset].Text = "╮"
				} else {
					plot[rows-y1][xAxis+config.Offset].Text = "╭"
					plot[rows-y0][xAxis+config.Offset].Text = "╯"
				}

				start := int(math.Min(float64(y0), float64(y1))) + 1
				end := int(math.Max(float64(y0), float64(y1)))
				for yAxis := start; yAxis < end; yAxis++ {
					plot[rows-yAxis][xAxis+config.Offset].Text = "│"
				}
			}

			start := int(math.Min(float64(y0), float64(y1)))
			end := int(math.Max(float64(y0), float64(y1)))
			for yAxis := start; yAxis <= end; yAxis++ {
				plot[rows-yAxis][xAxis+config.Offset].Color = color
			}
		}
	}

	// Render to screen
	rowsNum := rows + 1
	if config.HideAxisX {
		rowsNum--
	}
	for row := 0; row < rowsNum; row++ {
		// Find last non-space character in this row
		lastCharIndex := 0
		for i := width - 1; i >= 0; i-- {
			if plot[row][i].Text != " " {
				lastCharIndex = i
				break
			}
		}

		// Render each cell
		for col := 0; col <= lastCharIndex; col++ {
			cell := plot[row][col]
			style := tcell.StyleDefault.
				Foreground(cell.Color).
				Background(tcell.ColorDefault)

			runes := []rune(cell.Text)
			if len(runes) > 0 {
				screen.SetContent(x+col, y+row, runes[0], runes[1:], style)
			}
		}
	}

	// Render caption if not empty
	if config.Caption != "" {
		captionY := y + rows + 1
		captionX := x + config.Offset + maxWidth
		if len(config.Caption) < lenMax {
			captionX += (lenMax - len(config.Caption)) / 2
		}
		captionStyle := tcell.StyleDefault.
			Foreground(config.CaptionColor).
			Background(tcell.ColorDefault)
		for i, r := range []rune(config.Caption) {
			screen.SetContent(captionX+i, captionY, r, nil, captionStyle)
		}
	}

	// Render legends if present
	if len(config.SeriesLegends) > 0 {
		legendY := y + rows + 1
		if config.Caption != "" {
			legendY += 2
		}
		legendX := x + config.Offset + maxWidth

		var legendsTextLen int
		rightPad := 3
		for i, text := range config.SeriesLegends {
			color := tcell.ColorDefault
			if i < len(config.SeriesColors) {
				color = config.SeriesColors[i]
			}
			itemLen := createLegendItem(text, color)
			legendsTextLen += itemLen

			if i < len(config.SeriesLegends)-1 {
				legendsTextLen += rightPad
			}
		}

		if legendsTextLen < lenMax {
			legendX += (lenMax - legendsTextLen) / 2
		}

		// Render each legend item with its color
		col := legendX
		for i, text := range config.SeriesLegends {
			color := tcell.ColorDefault
			if i < len(config.SeriesColors) {
				color = config.SeriesColors[i]
			}
			legendStyle := tcell.StyleDefault.
				Foreground(color).
				Background(tcell.ColorDefault)

			// Render colored box
			screen.SetContent(col, legendY, '■', nil, legendStyle)
			col++

			// Render space
			screen.SetContent(col, legendY, ' ', nil, tcell.StyleDefault)
			col++

			// Render text in default color
			for _, r := range []rune(text) {
				screen.SetContent(col, legendY, r, nil, tcell.StyleDefault)
				col++
			}

			if i < len(config.SeriesLegends)-1 {
				for j := 0; j < rightPad; j++ {
					screen.SetContent(col, legendY, ' ', nil, tcell.StyleDefault)
					col++
				}
			}
		}
	}
}
