// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/gazed/vu"
	"github.com/gazed/vu/math/lin"
)

// cr, collision resolution, demonstrates simulated physics by having balls bounce
// on a floor. The neat thing is that after the initial locations have been set
// the physics simulation (vu/move) handles all subsequent position updates.
func cr() {
	cr := &crtag{}
	if err := vu.New(cr, "Collision Resolution", 400, 100, 800, 600); err != nil {
		log.Printf("cr: error initializing engine %s", err)
	}
	defer catchErrors()
}

// Globally unique "tag" that encapsulates example specific data.
type crtag struct {
	top     vu.Pov
	view    vu.View
	cam     vu.Camera
	striker vu.Pov  // Move to hit other items.
	run     float64 // Camera movement speed.
	spin    float64 // Camera spin speed.
}

// Create is the engine callback for initial asset creation.
func (cr *crtag) Create(eng vu.Eng, s *vu.State) {
	cr.run = 10   // move so many cubes worth in one second.
	cr.spin = 270 // spin so many degrees in one second.
	cr.top = eng.Root().NewPov()
	sun := cr.top.NewPov().SetLocation(0, 10, 10)
	sun.NewLight().SetColour(0.8, 0.8, 0.8)
	cr.view = cr.top.NewView()
	cr.cam = cr.view.Cam()
	cr.cam.SetPerspective(60, float64(800)/float64(600), 0.1, 500)
	cr.cam.SetLocation(0, 10, 25)

	// load the static slab.
	slab := cr.top.NewPov().SetScale(50, 50, 50).SetLocation(0, -25, 0)
	slab.NewBody(vu.NewBox(25, 25, 25))
	slab.SetSolid(0, 0.4)
	slab.NewModel("gouraud").LoadMesh("cube").LoadMat("floor")

	// create a single moving body.
	useBalls := true // Flip to use boxes instead of spheres.
	cr.striker = cr.top.NewPov()
	cr.striker.SetLocation(15, 15, 0) // -5, 15, -3
	if useBalls {
		cr.getBall(cr.striker)
	} else {
		cr.getBox(cr.striker)
		cr.striker.SetRotation(&lin.Q{X: 0.1825742, Y: 0.3651484, Z: 0.5477226, W: 0.7302967})
	}
	cr.striker.Model().SetColour(rand.Float64(), rand.Float64(), rand.Float64())

	// create a block of physics bodies.
	cubeSize := 3
	startX := -5 - cubeSize/2
	startY := -5
	startZ := -3 - cubeSize/2
	for k := 0; k < cubeSize; k++ {
		for i := 0; i < cubeSize; i++ {
			for j := 0; j < cubeSize; j++ {
				bod := cr.top.NewPov()
				lx := float64(2*i + startX)
				ly := float64(20 + 2*k + startY)
				lz := float64(2*j + startZ)
				bod.SetLocation(lx, ly, lz)
				if useBalls {
					cr.getBall(bod)
				} else {
					cr.getBox(bod)
				}
			}
		}
	}

	// set non default engine state.
	eng.SetColor(0.15, 0.15, 0.15, 1)
	rand.Seed(time.Now().UTC().UnixNano())
}

// Update is the regular engine callback.
func (cr *crtag) Update(eng vu.Eng, in *vu.Input, s *vu.State) {
	if in.Resized {
		cr.cam.SetPerspective(60, float64(s.W)/float64(s.H), 0.1, 50)
	}
	dt := in.Dt
	for press, _ := range in.Down {
		switch press {
		case "W":
			cr.cam.Move(0, 0, dt*-cr.run, cr.cam.Lookxz())
		case "S":
			cr.cam.Move(0, 0, dt*cr.run, cr.cam.Lookxz())
		case "A":
			cr.cam.AdjustYaw(dt * cr.spin)
		case "D":
			cr.cam.AdjustYaw(dt * -cr.spin)
		case "B":
			ball := cr.top.NewPov()
			ball.SetLocation(-2.5+rand.Float64(), 15, -1.5-rand.Float64())
			ball.NewBody(vu.NewSphere(1))
			ball.SetSolid(1, 0.9)
			m := ball.NewModel("gouraud").LoadMesh("sphere").LoadMat("sphere")
			m.SetColour(rand.Float64(), rand.Float64(), rand.Float64())
		case "Sp":
			body := cr.striker.Body()
			body.Push(-2.5, 0, -0.5)
		}
	}
}

// getBall creates a visible sphere physics body.
func (cr *crtag) getBall(p vu.Pov) {
	p.NewBody(vu.NewSphere(1))
	p.SetSolid(1, 0.5)
	p.NewModel("gouraud").LoadMesh("sphere").LoadMat("sphere")
}

// getBox creates a visible box physics body.
func (cr *crtag) getBox(p vu.Pov) {
	p.SetScale(2, 2, 2)
	p.NewBody(vu.NewBox(1, 1, 1))
	p.SetSolid(1, 0)
	p.NewModel("gouraud").LoadMesh("cube").LoadMat("sphere")
}