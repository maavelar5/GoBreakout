package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	gl "github.com/go-gl/gl/v3.0/gles2"
	"github.com/veandco/go-sdl2/sdl"
)

type vec2 struct {
	x, y float32
}

func (a vec2) Add(b vec2) vec2 {
	return vec2{a.x + b.x, a.y + b.y}
}

func (a vec2) Sub(b vec2) vec2 {
	return vec2{a.x - b.x, a.y - b.y}
}

func (a vec2) Mul(b vec2) vec2 {
	return vec2{a.x * b.x, a.y * b.y}
}

func (a *vec2) AddF(b float32) {
	a.x += b
	a.y += b
}

func (a vec2) Div(b vec2) vec2 {
	return vec2{a.x / b.x, a.y / b.y}
}

type vec3 struct {
	x, y, z float32
}

func (a vec3) Add(b vec3) vec3 {
	return vec3{a.x + b.x, a.y + b.y, a.z + b.z}
}

func (a vec3) Sub(b vec3) vec3 {
	return vec3{a.x - b.x, a.y - b.y, a.z - b.z}
}

func (a vec3) Mul(b vec3) vec3 {
	return vec3{a.x * b.x, a.y * b.y, a.z * b.z}
}

func (a vec3) Div(b vec3) vec3 {
	return vec3{a.x / b.x, a.y / b.y, a.z / b.z}
}

type vec4 struct {
	x, y, z, w float32
}

const (
	bit8  = 1
	bit16 = 2
	bit32 = 4
	bit64 = 8
)

func events(e sdl.Event) {
}

type Mat4 struct {
	data [4][4]float32
}

func Identity() Mat4 {
	return Mat4{
		[4][4]float32{
			{1, 0, 0, 0},
			{0, 1, 0, 0},
			{0, 0, 1, 0},
			{0, 0, 0, 1},
		},
	}
}

func (m1 Mat4) Mul(m2 Mat4) Mat4 {
	curr, result := float32(0), Mat4{}

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			curr = 0

			for k := 0; k < 4; k++ {
				curr += (m1.data[i][k] * m2.data[k][j])
			}

			result.data[i][j] = curr
		}
	}

	return result
}

func (m1 Mat4) ToArray() []float32 {
	result := []float32{}

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			result = append(result, m1.data[i][j])
		}
	}

	return result
}

type Ticks struct {
	paused              bool
	frames, fps, factor uint32

	current, previous, frame, step, accumulator, interpolation float32
}

func (t *Ticks) Init() {
	t.fps = 1
	t.step = 0.01
	t.factor = 1
	t.frames = 1
	t.paused = false
	t.previous = 0

	t.interpolation, t.accumulator, t.frame, t.current = 0, 0, 0, 0
}

func (t *Ticks) Update() {
	t.previous = t.current

	t.current = float32(sdl.GetTicks()) / 1000
	t.frame = t.current - t.previous

	if t.frame > 0.25 {
		t.frame = 0.25
	}

	t.accumulator += t.frame

	t.fps = uint32(float32(t.frames) / t.current)

	t.frames++
}

var ticks Ticks

const NONE uint32 = 0

const (
	SIMPLE  uint32 = 1
	TWO_WAY uint32 = 2
	LOOP    uint32 = 4
)

const (
	START        uint32 = 1
	WAIT         uint32 = 2
	DONE         uint32 = 4
	JUST_WAITED  uint32 = 8
	JUST_STARTED uint32 = 16
	JUST_DONE    uint32 = 32
	NO_PAUSE     uint32 = 64
)

type Timer struct {
	config, delay, restart_delay, current, state uint32
	loops                                        int32
}

func and(a uint32, b uint32) bool {
	if a&b == 0 {
		return false
	} else {
		return true
	}
}

func (t *Timer) Set(state uint32) {
	t.state = state
	t.current = sdl.GetTicks() * uint32(ticks.factor)
}

func (t *Timer) Update() {
	diff, just_set := (sdl.GetTicks()/ticks.factor)-t.current, NONE

	if ticks.paused && !and(t.config, NO_PAUSE) {
		t.current += uint32(ticks.frame)
		return
	}

	if t.state == NONE {
		t.Set(START | JUST_STARTED)
		just_set = JUST_STARTED
	} else if and(t.state, DONE) && and(t.config, LOOP) {
		t.Set(START)
	} else if and(t.state, START) && diff >= t.delay {
		if and(t.config, TWO_WAY) {
			t.Set(WAIT | JUST_WAITED)
			just_set = JUST_WAITED
		} else {
			t.Set(DONE | JUST_DONE)
			just_set = JUST_DONE
			t.loops++
		}
	} else if and(t.state, WAIT) && diff >= t.restart_delay {
		t.Set(DONE | JUST_DONE)
		just_set = JUST_DONE
	}

	if and(t.state, JUST_STARTED) && !and(just_set, JUST_STARTED) {
		t.state &= ^JUST_STARTED
	} else if and(t.state, JUST_WAITED) && !and(just_set, JUST_WAITED) {
		t.state &= ^JUST_WAITED
	} else if and(t.state, JUST_DONE) && !and(just_set, JUST_DONE) {
		t.state &= ^JUST_DONE
	}

}

type Shader struct {
	id, fId, vId, vao, vbo, ebo uint32
}

// TODO: OpenGL error handling
func CompileShaderFile(path string, shaderType uint32) uint32 {
	id := gl.CreateShader(shaderType)
	file, err := ioutil.ReadFile(path)

	if err != nil {
		panic(err)
	}

	source, free := gl.Strs(string(file) + "\x00")

	gl.ShaderSource(id, 1, source, nil)
	gl.CompileShader(id)

	free()

	return id
}

// TODO: OpenGL error handling
func CompileShader(vPath string, fPath string) Shader {
	s := Shader{
		gl.CreateProgram(),
		CompileShaderFile(vPath, gl.VERTEX_SHADER),
		CompileShaderFile(fPath, gl.FRAGMENT_SHADER),
		0, 0, 0,
	}

	gl.AttachShader(s.id, s.vId)
	gl.AttachShader(s.id, s.fId)

	gl.LinkProgram(s.id)

	gl.GenVertexArrays(1, &s.vao)
	gl.GenBuffers(1, &s.vbo)
	gl.GenBuffers(1, &s.ebo)

	gl.BindVertexArray(s.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.ebo)

	return s
}

func Ortho(W float32, H float32) Mat4 {
	r, t := W, float32(0)
	l, b := float32(0), H
	f, n := float32(1), float32(-1)

	matrix := Identity()

	matrix.data[0][0] = 2 / (r - l)
	matrix.data[0][3] = -((r + l) / (r - l))

	matrix.data[1][1] = 2 / (t - b)
	matrix.data[1][3] = -((t + b) / (t - b))

	matrix.data[2][2] = -2 / (f - n)
	matrix.data[2][3] = -((f + n) / (f - n))

	return matrix
}

func (m1 Mat4) Translate(pos vec2) Mat4 {
	transMatrix := Identity()

	// {1, 0, 0, pos.x}
	// {0, 1, 0, pos.y}
	// {0, 0, 1, 0    }
	// {0, 0, 0, 1    }

	transMatrix.data[0][3] = pos.x
	transMatrix.data[1][3] = pos.y

	return m1.Mul(transMatrix)
}

func (m Mat4) Scale(size vec3) Mat4 {
	scaleMatrix := Identity()

	// {size.x, 0,      0,      0}
	// {0,      size.y, 0,      0}
	// {0,      0,      size.z, 0}
	// {0,      0,      0,      1}

	scaleMatrix.data[0][0] = size.x
	scaleMatrix.data[1][1] = size.y
	scaleMatrix.data[2][2] = size.z

	return m.Mul(scaleMatrix)
}

func (s Shader) Location(str string) int32 {
	return gl.GetUniformLocation(s.id, gl.Str(str+"\x00"))
}

func (s Shader) SetVec4(str string, v vec4) {
	raw := [4]float32{v.x, v.y, v.z, v.w}

	gl.Uniform4fv(s.Location(str), 1, (*float32)(gl.Ptr(&raw[0])))
}

func (s Shader) SetMat4(str string, m Mat4) {
	gl.UniformMatrix4fv(s.Location(str), 1, true, (*float32)(gl.Ptr(m.ToArray())))
}

func (s Shader) Use() {
	gl.Flush()

	gl.UseProgram(s.id)
	gl.BindVertexArray(s.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, s.vbo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, s.ebo)
}

func getModel(pos vec2, size vec2) Mat4 {
	m := Identity()

	m = m.Translate(pos)
	m = m.Scale(vec3{size.x, size.y, 0})

	return m
}

func defaultShader() Shader {
	s := CompileShader("vertex.glsl", "fragment.glsl")

	points := []float32{
		0, 0, //
		0, 1, //
		1, 1, //
		1, 0, //
	}

	indices := []int32{
		0, 1, 2, //
		0, 2, 3, //
	}

	gl.UseProgram(s.id)

	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*int32(bit32), nil)
	gl.EnableVertexAttribArray(0)

	gl.BufferData(gl.ARRAY_BUFFER, len(points)*bit32, gl.Ptr(points), gl.STATIC_DRAW)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*bit32, gl.Ptr(indices), gl.STATIC_DRAW)

	s.SetVec4("uColor", vec4{0, 1, 0, 1})

	s.SetMat4("uModel", Identity())
	s.SetMat4("uProjection", Ortho(320, 180))

	return s
}

func defaultShaderBatch() Shader {
	s := CompileShader("vertex.glsl", "fragment.glsl")

	points := [4 * (10000 * (2 + 16 + 4))]float32{}

	indices := []int32{}

	//NOTE: vec2, Mat4, vec4
	stride := int32((2 + 16 + 4) * bit32)

	for i := int32(0); i < int32(len(points)); i += 4 {
		indices = append(indices, i+0, i+1, i+2, i+1, i+2, i+3)
	}

	s.Use()

	//aVec
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	//aModel1
	gl.VertexAttribPointer(1, 4, gl.FLOAT, false, stride, gl.PtrOffset(2*bit32))
	gl.EnableVertexAttribArray(1)

	//aModel2
	gl.VertexAttribPointer(2, 4, gl.FLOAT, false, stride, gl.PtrOffset(6*bit32))
	gl.EnableVertexAttribArray(2)

	//aModel3
	gl.VertexAttribPointer(3, 4, gl.FLOAT, false, stride, gl.PtrOffset(10*bit32))
	gl.EnableVertexAttribArray(3)

	//aModel4
	gl.VertexAttribPointer(4, 4, gl.FLOAT, false, stride, gl.PtrOffset(14*bit32))
	gl.EnableVertexAttribArray(4)

	//aColor
	gl.VertexAttribPointer(5, 4, gl.FLOAT, false, stride, gl.PtrOffset(18*bit32))
	gl.EnableVertexAttribArray(5)

	gl.BufferData(gl.ARRAY_BUFFER, len(points)*bit32, gl.Ptr(&points[0]), gl.STREAM_DRAW)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*bit32, gl.Ptr(indices), gl.STATIC_DRAW)

	s.SetMat4("uProjection", Ortho(320, 180))

	return s
}

func randf(val float32) float32 {
	return rand.Float32() * val
}

func randVec2(x float32, y float32) vec2 {
	return vec2{
		float32(rand.Intn(int(x))),
		float32(rand.Intn(int(y))),
	}
}

func randVec4() vec4 {
	return vec4{
		rand.Float32(),
		rand.Float32(),
		rand.Float32(),
		rand.Float32(),
	}
}

type Body struct {
	pos, size vec2
	angle     float32
	touched   bool
}

type Block struct {
	Body
	color vec4
}

func PushModel(b Block, data *[]float32, drawCalls *int32) {
	model := getModel(b.pos, b.size)
	pointReference := [8]float32{
		0, 0, //
		1, 0, //
		0, 1, //
		1, 1, //
	}

	for j := 0; j < len(pointReference); j += 2 {
		*data = append(*data, pointReference[j], pointReference[j+1])

		for k := 0; k < len(model.data[0]); k++ {
			for l := 0; l < len(model.data[k]); l++ {
				*data = append(*data, model.data[l][k])
			}
		}

		*data = append(*data, b.color.x, b.color.y, b.color.z, b.color.w)
	}

	*drawCalls += 6
}

func CheckAxis(a Body, b Body) bool {
	if (a.pos.x < b.pos.x && a.pos.x+a.size.x < b.pos.x) || a.pos.x > b.pos.x+b.size.x {
		return false
	} else if (a.pos.y < b.pos.y && a.pos.y+a.size.y < b.pos.y) || a.pos.y > b.pos.y+b.size.y {
		return false
	} else {
		return true
	}
}

func CheckCollision(a *Body, b *Body) bool {
	if CheckAxis(*a, *b) {
		a.touched = true
		b.touched = true

		return true
	} else {
		return false
	}
}

func InitBlocks() []Block {
	blocks := []Block{}

	for i := float32(0); i < 10; i++ {
		for j := float32(0); j < 10; j++ {
			block := Block{Body{vec2{35 + (i * 25), 20 + (j * 8)}, vec2{24, 4}, 0, false}, randVec4()}
			block.color.w = 1.0

			blocks = append(blocks, block)
		}
	}

	return blocks
}

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}

	const W, H = float32(320), float32(180)

	rand.Seed(time.Now().UnixNano())

	ticks.Init()

	defer sdl.Quit()

	window, err := sdl.CreateWindow("cardeyb", 0, 0, 640, 360, sdl.WINDOW_SHOWN|sdl.WINDOW_OPENGL|sdl.WINDOW_FULLSCREEN_DESKTOP)

	if err != nil {
		panic(err)
	}

	context, err := window.GLCreateContext()

	if err != nil {
		panic(err)
	}

	sdl.GLSetSwapInterval(0)

	defer sdl.GLDeleteContext(context)
	defer window.Destroy()

	if err = gl.Init(); err != nil {
		panic(err)
	}

	running := true

	window.SetWindowOpacity(1)

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	timer := Timer{TWO_WAY | LOOP, 2000, 2000, 0, 0, 0}

	s := defaultShaderBatch()

	s.Use()

	ball := Block{Body{vec2{W/2 - 2, H - 20}, vec2{8, 8}, 0, false}, vec4{0, 1, 0, 1}}
	bodies := []Block{}
	player := Block{Body{vec2{W/2 - 12, H - 10}, vec2{24, 4}, 0, false}, vec4{1, 1, 1, 1}}
	playerVel := float32(0)
	playerSpeed := float32(200)

	ballVelY := float32(75)
	ballVelX := float32(75)
	ballSpeed := float32(75)

	bodies = InitBlocks()

	for running {
		ticks.Update()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
				break
			case *sdl.KeyboardEvent:
				if t.Type == sdl.KEYDOWN && t.Repeat == 0 {
					if t.Keysym.Sym == sdl.K_a {
						playerVel -= playerSpeed
					} else if t.Keysym.Sym == sdl.K_d {
						playerVel += playerSpeed
					}
				} else if t.Type == sdl.KEYUP {
					if t.Keysym.Sym == sdl.K_a {
						playerVel += playerSpeed
					} else if t.Keysym.Sym == sdl.K_d {
						playerVel -= playerSpeed
					}
				}

				break
			}
		}

		for ticks.accumulator > ticks.step {
			ball.pos.x += ballVelX * ticks.step
			ball.pos.y += ballVelY * ticks.step

			player.pos.x += playerVel * ticks.step

			if CheckCollision(&player.Body, &ball.Body) {
				if playerVel > 0 {
					ballVelX = ballSpeed
				} else if playerVel < 0 {
					ballVelX = -ballSpeed
				}
			}

			for i := 0; i < len(bodies); i++ {
				if CheckCollision(&ball.Body, &bodies[i].Body) {
					bodies = append(bodies[:i], bodies[i+1:]...)
					break
				}
			}

			if ball.touched {
				ballVelY = -ballVelY
			}

			if ball.pos.x <= 0 {
				ballVelX = ballSpeed
			} else if ball.pos.x+ball.size.x >= W {
				ballVelX = -ballSpeed
			}

			if ball.pos.y <= 0 {
				ballVelY = ballSpeed
			} else if ball.pos.y+ball.size.y >= H {
				ballVelY = -ballSpeed
			}

			ball.touched = false

			ticks.accumulator -= ticks.step
		}

		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		timer.Update()

		if and(timer.state, JUST_STARTED) {
			fmt.Println("JUST_STARTED")
		} else if and(timer.state, JUST_WAITED) {
			fmt.Println("JUST_WAITED")
		} else if and(timer.state, JUST_DONE) {
			fmt.Println("JUST_DONE")
			fmt.Println(ticks.fps)
		}

		data := []float32{}
		drawCalls := int32(0)

		PushModel(player, &data, &drawCalls)
		PushModel(ball, &data, &drawCalls)

		for _, body := range bodies {
			PushModel(body, &data, &drawCalls)
		}

		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*bit32, gl.Ptr(data))
		gl.DrawElements(gl.TRIANGLES, drawCalls, gl.UNSIGNED_INT, gl.PtrOffset(0))

		if len(bodies) == 0 {
			bodies = InitBlocks()
		}

		window.GLSwap()
	}
}
