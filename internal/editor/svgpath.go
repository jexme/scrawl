package editor

import (
	"image/color"
	"math"
	"strconv"
	"unicode"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// segment — одна команда пути в абсолютных координатах.
type segment struct {
	kind byte // 'M', 'L', 'C', 'Z'
	pts  [3]f32.Point
}

// parseSVGPath разбирает атрибут d из SVG: M L H V C S A Z в абсолютной и
// относительной формах (этого хватает для иконок Tabler). Дуги превращаются в
// кубические кривые.
func parseSVGPath(d string) []segment {
	l := &pathLexer{s: d}
	var (
		segs       []segment
		cur, start f32.Point
		lastCtrl   f32.Point
		prevCubic  bool
		cmd        byte
		moveToLine = map[byte]byte{'M': 'L', 'm': 'l'}
	)
	for {
		l.skipSep()
		if l.eof() {
			break
		}
		if c := l.peek(); unicode.IsLetter(rune(c)) {
			cmd = c
			l.pos++
		} else if cmd == 0 {
			break
		} else if cmd == 'Z' || cmd == 'z' {
			break
		}
		rel := cmd >= 'a'
		up := cmd &^ 0x20
		pt := func(x, y float32) f32.Point {
			if rel {
				return f32.Pt(cur.X+x, cur.Y+y)
			}
			return f32.Pt(x, y)
		}

		switch up {
		case 'M':
			p := pt(l.num(), l.num())
			segs = append(segs, segment{kind: 'M', pts: [3]f32.Point{p}})
			cur, start, prevCubic = p, p, false
			if next, ok := moveToLine[cmd]; ok {
				cmd = next // повторные пары после M означают L
			}
		case 'L':
			p := pt(l.num(), l.num())
			segs = append(segs, segment{kind: 'L', pts: [3]f32.Point{p}})
			cur, prevCubic = p, false
		case 'H':
			x := l.num()
			if rel {
				x += cur.X
			}
			p := f32.Pt(x, cur.Y)
			segs = append(segs, segment{kind: 'L', pts: [3]f32.Point{p}})
			cur, prevCubic = p, false
		case 'V':
			y := l.num()
			if rel {
				y += cur.Y
			}
			p := f32.Pt(cur.X, y)
			segs = append(segs, segment{kind: 'L', pts: [3]f32.Point{p}})
			cur, prevCubic = p, false
		case 'C':
			c1 := pt(l.num(), l.num())
			c2 := pt(l.num(), l.num())
			p := pt(l.num(), l.num())
			segs = append(segs, segment{kind: 'C', pts: [3]f32.Point{c1, c2, p}})
			cur, lastCtrl, prevCubic = p, c2, true
		case 'S':
			c1 := cur
			if prevCubic {
				c1 = f32.Pt(2*cur.X-lastCtrl.X, 2*cur.Y-lastCtrl.Y)
			}
			c2 := pt(l.num(), l.num())
			p := pt(l.num(), l.num())
			segs = append(segs, segment{kind: 'C', pts: [3]f32.Point{c1, c2, p}})
			cur, lastCtrl, prevCubic = p, c2, true
		case 'A':
			rx, ry, rot := l.num(), l.num(), l.num()
			fa, fs := l.flag(), l.flag()
			p := pt(l.num(), l.num())
			segs = append(segs, arcToCubics(cur, rx, ry, rot, fa, fs, p)...)
			cur, prevCubic = p, false
		case 'Z':
			segs = append(segs, segment{kind: 'Z'})
			cur, prevCubic = start, false
		default:
			return segs
		}
	}
	return segs
}

// pathLexer читает числа и флаги из строки пути.
type pathLexer struct {
	s   string
	pos int
}

func (l *pathLexer) eof() bool  { return l.pos >= len(l.s) }
func (l *pathLexer) peek() byte { return l.s[l.pos] }

func (l *pathLexer) skipSep() {
	for !l.eof() && (l.s[l.pos] == ' ' || l.s[l.pos] == ',' || l.s[l.pos] == '\n' || l.s[l.pos] == '\t') {
		l.pos++
	}
}

func (l *pathLexer) num() float32 {
	l.skipSep()
	start := l.pos
	if !l.eof() && (l.s[l.pos] == '-' || l.s[l.pos] == '+') {
		l.pos++
	}
	seenDot := false
	for !l.eof() {
		c := l.s[l.pos]
		if c >= '0' && c <= '9' {
			l.pos++
		} else if c == '.' && !seenDot {
			seenDot = true
			l.pos++
		} else {
			break
		}
	}
	v, _ := strconv.ParseFloat(l.s[start:l.pos], 32)
	return float32(v)
}

// flag читает флаг дуги: один символ 0 или 1.
func (l *pathLexer) flag() bool {
	l.skipSep()
	if l.eof() {
		return false
	}
	c := l.s[l.pos]
	l.pos++
	return c == '1'
}

// arcToCubics переводит дугу SVG (по конечным точкам) в кубические кривые.
func arcToCubics(from f32.Point, rx, ry, rotDeg float32, large, sweep bool, to f32.Point) []segment {
	x1, y1, x2, y2 := float64(from.X), float64(from.Y), float64(to.X), float64(to.Y)
	a, b := math.Abs(float64(rx)), math.Abs(float64(ry))
	if a == 0 || b == 0 || (x1 == x2 && y1 == y2) {
		return []segment{{kind: 'L', pts: [3]f32.Point{to}}}
	}
	phi := float64(rotDeg) * math.Pi / 180
	cosP, sinP := math.Cos(phi), math.Sin(phi)

	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p := cosP*dx + sinP*dy
	y1p := -sinP*dx + cosP*dy

	if lam := x1p*x1p/(a*a) + y1p*y1p/(b*b); lam > 1 {
		s := math.Sqrt(lam)
		a, b = a*s, b*s
	}
	num := a*a*b*b - a*a*y1p*y1p - b*b*x1p*x1p
	den := a*a*y1p*y1p + b*b*x1p*x1p
	co := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		co = -co
	}
	cxp := co * a * y1p / b
	cyp := -co * b * x1p / a
	cx := cosP*cxp - sinP*cyp + (x1+x2)/2
	cy := sinP*cxp + cosP*cyp + (y1+y2)/2

	angle := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	theta1 := angle(1, 0, (x1p-cxp)/a, (y1p-cyp)/b)
	dtheta := angle((x1p-cxp)/a, (y1p-cyp)/b, (-x1p-cxp)/a, (-y1p-cyp)/b)
	if !sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	} else if sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	n := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	delta := dtheta / float64(n)
	t := 4.0 / 3.0 * math.Tan(delta/4)
	mapPt := func(px, py float64) f32.Point {
		return f32.Pt(float32(cx+a*px*cosP-b*py*sinP), float32(cy+a*px*sinP+b*py*cosP))
	}
	var out []segment
	for i := 0; i < n; i++ {
		t1 := theta1 + float64(i)*delta
		t2 := t1 + delta
		c1 := mapPt(math.Cos(t1)-t*math.Sin(t1), math.Sin(t1)+t*math.Cos(t1))
		c2 := mapPt(math.Cos(t2)+t*math.Sin(t2), math.Sin(t2)-t*math.Cos(t2))
		p := mapPt(math.Cos(t2), math.Sin(t2))
		if i == n-1 {
			p = to // без накопленной погрешности
		}
		out = append(out, segment{kind: 'C', pts: [3]f32.Point{c1, c2, p}})
	}
	return out
}

// buildPath строит clip.Path из SVG-пути: масштаб scale на единицу сетки, смещение off.
func buildPath(ops *op.Ops, d string, off f32.Point, scale float32) clip.PathSpec {
	var p clip.Path
	p.Begin(ops)
	tf := func(pt f32.Point) f32.Point { return f32.Pt(off.X+pt.X*scale, off.Y+pt.Y*scale) }
	for _, s := range parseSVGPath(d) {
		switch s.kind {
		case 'M':
			p.MoveTo(tf(s.pts[0]))
		case 'L':
			p.LineTo(tf(s.pts[0]))
		case 'C':
			p.CubeTo(tf(s.pts[0]), tf(s.pts[1]), tf(s.pts[2]))
		case 'Z':
			p.Close()
		}
	}
	return p.End()
}

// strokeSVG рисует контур по SVG-пути толщиной width пикселей.
func strokeSVG(ops *op.Ops, c color.NRGBA, d string, off f32.Point, scale, width float32) {
	paint.FillShape(ops, c, clip.Stroke{Path: buildPath(ops, d, off, scale), Width: width}.Op())
}

// fillSVG заливает замкнутый SVG-путь.
func fillSVG(ops *op.Ops, c color.NRGBA, d string, off f32.Point, scale float32) {
	paint.FillShape(ops, c, clip.Outline{Path: buildPath(ops, d, off, scale)}.Op())
}
