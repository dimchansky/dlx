package main

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// point is a corner of a kite. Its coordinates count steps of length 1 along the
// directions at 0 and 60 degrees, so every corner has whole coordinates.
type point struct {
	a, b int
}

// hatOutline is the boundary of the hat, as published by Kaplan in the source
// of the hatviz program by the authors of "An aperiodic monotile". In this
// frame a hexagon of the kite grid is centred on the origin.
var hatOutline = [...]point{
	{0, 0},
	{-1, -1},
	{0, -2},
	{2, -2},
	{2, -1},
	{4, -2},
	{5, -1},
	{4, 0},
	{3, 0},
	{2, 2},
	{0, 3},
	{0, 2},
	{-1, 2},
}

// outline returns the boundary of p, applying the same steps as kites.
func (p placement) outline() []point {
	corners := make([]point, len(hatOutline))
	for i, corner := range hatOutline {
		if p.mirrored {
			corner = point{corner.b, corner.a}
		}
		for step := 0; step < p.turns; step++ {
			corner = point{-corner.b, corner.a + corner.b}
		}
		corners[i] = point{corner.a + 2*p.q - 2*p.r, corner.b + 2*p.q + 4*p.r}
	}
	return corners
}

// corners returns the four corners of k: the middle of its hexagon, the middle
// of one side, the corner of the hexagon, and the middle of the next side.
func (k kite) corners() [4]point {
	directions := [turns]point{{1, 0}, {0, 1}, {-1, 1}, {-1, 0}, {0, -1}, {1, -1}}
	middle := point{2*k.q - 2*k.r, 2*k.q + 4*k.r}
	before := directions[(k.dir+turns-1)%turns]
	tip := directions[k.dir]
	after := directions[(k.dir+1)%turns]
	return [4]point{
		middle,
		{middle.a + before.a + tip.a, middle.b + before.b + tip.b},
		{middle.a + 2*tip.a, middle.b + 2*tip.b},
		{middle.a + tip.a + after.a, middle.b + tip.b + after.b},
	}
}

const (
	svgScale  = 12
	svgMargin = 2
	// The white card keeps the picture readable on light and dark pages.
	svgInk         = "#24292f"
	svgKite        = "#57606a"
	svgUnreflected = "#a8d5ff"
	svgReflected   = "#ffb347"
	svgDisk        = "#d1242f"
)

// writeSVG draws the hats and the disk, on a white card. Reflected hats are
// orange and unreflected hats are blue.
func writeSVG(w io.Writer, hats []placement, radius int) error {
	half := float64(radius)
	for _, hat := range hats {
		for _, corner := range hat.outline() {
			x, y := corner.position()
			half = math.Max(half, math.Max(math.Abs(x), math.Abs(y)))
		}
	}
	half += svgMargin

	size := 2 * half * svgScale
	label := fmt.Sprintf("%d hats covering a disk of radius %d, %d of them reflected",
		len(hats), radius, countReflected(hats))

	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.0f %.0f" width="%.0f" height="%.0f" role="img" aria-label="%s">`+"\n",
		size, size, size, size, label)
	fmt.Fprintf(&svg, "<title>%s</title>\n", label)
	fmt.Fprintf(&svg, `<rect x="0.5" y="0.5" width="%.0f" height="%.0f" rx="8" fill="#ffffff" stroke="#8c959f"/>`+"\n", size-1, size-1)
	for _, hat := range hats {
		fill := svgUnreflected
		if hat.mirrored {
			fill = svgReflected
		}
		for _, k := range hat.kites() {
			corners := k.corners()
			fmt.Fprintf(&svg, `<polygon points="%s" fill="%s" stroke="%s" stroke-width="0.5"/>`+"\n",
				svgPoints(corners[:], half), fill, svgKite)
		}
		fmt.Fprintf(&svg, `<polygon points="%s" fill="none" stroke="%s" stroke-width="1.5" stroke-linejoin="round"/>`+"\n",
			svgPoints(hat.outline(), half), svgInk)
	}
	fmt.Fprintf(&svg, `<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="%s" stroke-width="2" stroke-dasharray="6 4"/>`+"\n",
		half*svgScale, half*svgScale, float64(radius)*svgScale, svgDisk)
	svg.WriteString("</svg>\n")

	_, err := io.WriteString(w, svg.String())
	return err
}

// svgPoints lists corners in picture coordinates, where y points down.
func svgPoints(corners []point, half float64) string {
	var list []string
	for _, corner := range corners {
		x, y := corner.position()
		list = append(list, fmt.Sprintf("%.1f,%.1f", (x+half)*svgScale, (half-y)*svgScale))
	}
	return strings.Join(list, " ")
}

// position returns the point on a plane with the y axis pointing up.
func (p point) position() (x, y float64) {
	return float64(p.a) + float64(p.b)/2, float64(p.b) * math.Sqrt(3) / 2
}
