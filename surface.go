// Copyright © 2014-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package vu

import (
	"math"

	"github.com/gazed/vu/render"
)

// Surface renders land height data. The surface is rendered
// based on the height and texture index information in SurfacePoints.
// Surface populates a render Models mesh data.
type Surface interface {
	Pts() [][]SurfacePoint      // Per vertex information.
	Update(m Model, xo, yo int) // Generates rendering data into Model.
	Resize(w, h int)            // Resize the surface point holders.
}

// SurfacePoint stores a height value and a texture atlas index
// for one point in a Surface. Blend indicates the amount of blending of
// the texture at the given index with the next texture in the atlas.
type SurfacePoint struct {
	Height float32 // Surface height value.
	Tindex int     // Surface texture atlas index.
	Blend  float32 // Texture blend value between 0 and 1.
}

// NewSurface creates a surface that holds a sx-by-sy set of SurfacePoints.
//    spread       : the number of tiles one texture covers.
//    textureRatio : the size of one texture to the size of the texture atlas.
//    scale        : the amount of scaling applied to each height.
func NewSurface(sx, sy, spread int, textureRatio, scale float32) Surface {
	return newSurface(sx, sy, spread, textureRatio, scale)
}

// Surface
// ============================================================================
// surface implements Surface.

// surface is used to render land height data. The surface is rendered
// A surface model relies on and is expected to be combined with:
//    vu/land : to create the height map.
//    images/ : for a texture atlas resource.
//    source/ : for a surface specific shader.
type surface struct {
	tratio float32          // Texture atlas ratio (textureSize/atlasSize).
	scale  float32          // Height scaling factor.
	spread int              // Smear texture across tiles. 1, 2, 4, 8, ...
	pts    [][]SurfacePoint // Per vertex information.

	// scratch rendering data. Reused each time Update is called.
	vb  []float32 // Scratch vertex buffer
	nb  []float32 // Scratch normal buffer
	tb  []float32 // Scratch texture uv buffer
	fb  []uint16  // Scratch face buffer
	nms [][]xyz   // Scratch for normal calculations.
}

// newSurface allocates and initializes surface.
func newSurface(sx, sy, spread int, textureRatio, scale float32) *surface {
	s := &surface{}
	s.tratio = textureRatio
	s.spread = spread
	s.scale = scale
	s.pts = make([][]SurfacePoint, sx)
	for x := range s.pts {
		s.pts[x] = make([]SurfacePoint, sy)
	}
	s.vb = []float32{}
	s.nb = []float32{}
	s.tb = []float32{}
	s.fb = []uint16{}

	// scratch for normal generation.
	s.nms = make([][]xyz, len(s.pts))
	for x := range s.nms {
		s.nms[x] = make([]xyz, len(s.pts[0]))
	}
	return s
}

// Implement Surface.
func (s *surface) Pts() [][]SurfacePoint { return s.pts }
func (s *surface) Resize(w, h int) {
	s.pts = s.pts[:w]
	for sx := range s.pts {
		s.pts[sx] = s.pts[sx][:h]
	}
}

// Update recalculates the vertex data needed to render the given land patch.
// It also uses the texture index to assign a textures from a texture atlas
func (s *surface) Update(m Model, xoff, yoff int) {
	vb := s.vb[:0] // keep any allocated memory.
	nb := s.nb[:0] //   "
	tb := s.tb[:0] //   "
	fb := s.fb[:0] //   "

	// generate the per-vertex normals based on the slopes to connecting verticies.
	// http://www.flipcode.com/archives/Calculating_Vertex_Normals_for_Height_Maps.shtml
	// http://www.gamedev.net/topic/163625-fast-way-to-calculate-heightmap-normals/
	sx, sy := len(s.pts), len(s.pts[0])
	norms := s.nms
	yScale, xzScale := s.scale, float32(1)
	for x := 0; x < sx; x++ {
		for y := 0; y < sy; y++ {

			// average xslope
			xmax, xmin := x, x
			if xmax < sx-1 {
				xmax++
			}
			if xmin > 0 {
				xmin--
			}
			xslope := float32(s.pts[xmax][y].Height - s.pts[xmin][y].Height)
			if x == 0 || x == sx-1 {
				xslope *= 2
			}

			// average yslope
			ymax, ymin := y, y
			if ymax < sy-1 {
				ymax++
			}
			if ymin > 0 {
				ymin--
			}
			yslope := float32(s.pts[x][ymax].Height - s.pts[x][ymin].Height)
			if y == 0 || y == sy-1 {
				yslope *= 2
			}

			// store the unit length normal.
			nx, ny, nz := -xslope*yScale, 2*xzScale, yslope*yScale
			length := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
			norms[x][y].x, norms[x][y].y, norms[x][y].z = nx/length, ny/length, nz/length
		}
	}

	// UV texture coordinate values.
	textureRatio := s.tratio                  // single texture to texture atlas value.
	width := textureRatio / float32(s.spread) // tile width.
	border := float32(0.001)

	// Generate the verticies, triangle faces, and matching normals.
	hscale := s.scale // scaling range of 1 to -1
	vc := uint16(0)   // vertex counter.
	for x := 0; x < sx-1; x++ {
		for y := 0; y < sy-1; y++ {

			// Generate the verticies for one quad.
			vx0, vy0, vz0 := float32(x), float32(y), s.pts[x][y].Height*hscale
			vx1, vy1, vz1 := float32(x+1), float32(y), s.pts[x+1][y].Height*hscale
			vx2, vy2, vz2 := float32(x), float32(y+1), s.pts[x][y+1].Height*hscale
			vx3, vy3, vz3 := float32(x+1), float32(y+1), s.pts[x+1][y+1].Height*hscale
			vb = append(vb, vx0, vy0, vz0)
			vb = append(vb, vx1, vy1, vz1)
			vb = append(vb, vx2, vy2, vz2)
			vb = append(vb, vx3, vy3, vz3)

			// Pack the uv indicies with the texture index and blend factor.
			basex := float32((x+xoff)%s.spread) / float32(s.spread)
			basey := 1.0 - float32((y+yoff)%s.spread)/float32(s.spread) - 1/float32(s.spread)
			uv0, uv1 := basex*textureRatio, basey*textureRatio+width       // uv0 top-left     0,1
			uv2, uv3 := basex*textureRatio+width, basey*textureRatio+width // uv1 top-right    1,1
			uv4, uv5 := basex*textureRatio, basey*textureRatio             // uv3 bottom-left  0,0
			uv6, uv7 := basex*textureRatio+width, basey*textureRatio       // uv4 bottom-right 1,0

			// Add a small border to the outside of the overall texture
			// to avoid a white line between textures.
			if uv0 == 0 {
				uv0 += border
				uv4 += border
			}
			if uv2 == textureRatio {
				uv2 -= border
				uv6 -= border
			}
			if uv5 == 0 {
				uv5 += border
				uv7 += border
			}
			if uv1 == textureRatio {
				uv1 -= border
				uv3 -= border
			}
			tindex, blend := float32(s.pts[x][y].Tindex), s.pts[x][y].Blend
			tb = append(tb, uv0, uv1, tindex, blend)
			tb = append(tb, uv2, uv3, tindex, blend)
			tb = append(tb, uv4, uv5, tindex, blend)
			tb = append(tb, uv6, uv7, tindex, blend)

			// Generate the triangle faces for the above quad.
			fb = append(fb, vc, vc+1, vc+2, vc+1, vc+3, vc+2)
			vc += 4

			// Add normal information for each vertex in the map quad.
			nb = append(nb, norms[x][y].x, norms[x][y].y, norms[x][y].z)
			nb = append(nb, norms[x+1][y].x, norms[x+1][y].y, norms[x+1][y].z)
			nb = append(nb, norms[x][y+1].x, norms[x][y+1].y, norms[x][y+1].z)
			nb = append(nb, norms[x+1][y+1].x, norms[x+1][y+1].y, norms[x+1][y+1].z)
		}
	}
	m.InitMesh(0, 3, render.DynamicDraw, false).SetMeshData(0, vb)
	m.InitMesh(1, 3, render.DynamicDraw, false).SetMeshData(1, nb)
	m.InitMesh(2, 4, render.DynamicDraw, false).SetMeshData(2, tb)
	m.InitFaces(render.DynamicDraw).SetFaces(fb)
}

type xyz struct{ x, y, z float32 } // temporary structure for generating normals.
