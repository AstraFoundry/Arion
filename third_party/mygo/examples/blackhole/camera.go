package main

import "math"

// vec3 is a point or a direction of the scene, y up, in radii of the
// hole.
type vec3 [3]float64

func (a vec3) add(b vec3) vec3    { return vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a vec3) mul(k float64) vec3 { return vec3{a[0] * k, a[1] * k, a[2] * k} }
func (a vec3) dot(b vec3) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a vec3) norm() vec3         { return a.mul(1 / math.Sqrt(a.dot(a))) }
func (a vec3) f32() [3]float32    { return [3]float32{float32(a[0]), float32(a[1]), float32(a[2])} }
func (a vec3) cross(b vec3) vec3 {
	return vec3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

// focal is the camera's focal length, in heights of the view.
const focal = 1.5

// camera orbits the hole: its angles about the disk's axis (yaw) and above
// the disk (pitch), in radians, its distance, and how fast the angles turn
// on their own, after a drag's throw.
type camera struct {
	yaw, pitch, dist   float64
	spinYaw, spinPitch float64
}

// axes returns where the camera is and its right, up and forward axes.
func (c *camera) axes() (eye, right, up, forward vec3) {
	cp := math.Cos(c.pitch)
	eye = vec3{cp * math.Sin(c.yaw), math.Sin(c.pitch), cp * math.Cos(c.yaw)}.mul(c.dist)
	forward = eye.mul(-1).norm()
	right = forward.cross(vec3{0, 1, 0}).norm()
	up = right.cross(forward)
	return
}
