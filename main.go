package main

import (
	"embed"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v3.2-compatibility/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

//go:embed noto.ttf
var f embed.FS

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newTexture(file string) (uint32, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return 0, fmt.Errorf("texture %q not found on disk: %v", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}

func newTextureRGBA(rgba *image.RGBA) (uint32, error) {
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}

var windowWidth int = 800
var windowHeight int = 600

var DotVertexShader = `
#version 330
const float PI = 3.14159265359;
layout(location = 0) in vec3 vert;
uniform vec2 window_size;
uniform mat3 scale;
uniform float time;

void main() {
	mat4 window_scale = mat4 (
		vec4(window_size.x/window_size.y, 0.0, 0.0, 0.0),
		vec4(    0.0, 1.0, 0.0, 0.0),
		vec4(    0.0, 0.0, 1.0, 0.0),
		vec4(    0.0, 0.0, 0.0, 1.0)
	);
	float base = time * 5.0 + 1.0 * gl_InstanceID;
	float offset = PI * (abs((((base/5)-1)/2) - (floor(base/5/2))) + ((base/5-1)));
	mat4 transform = mat4 (
		vec4(1.0, 0.0, 0.0, sin(offset)*0.09),
		vec4(0.0, 1.0, 0.0, cos(offset)*0.09+0.2),
		vec4(0.0, 0.0, 1.0, 0.0),
		vec4(0.0, 0.0, 0.0, 1.0)
	);
	gl_Position = vec4(vert * scale, 1.0) * transform * window_scale;
}
` + "\x00"

var DotFragmentShader = `
#version 330
uniform sampler2D tex;
in vec2 fragTexCoord;
out vec4 outputColor;
void main() {
    outputColor = vec4(1,1,1, 1);
}
` + "\x00"

var TextVertexShader = `
#version 330
uniform vec2 window_size;
in vec3 vert;
in vec2 vertTexCoord;
out vec2 fragTexCoord;

void main() {
	mat4 window_scale = mat4 (
		vec4(window_size.x/window_size.y, 0.0, 0.0, 0.0),
		vec4(0.0, 1.0, 0.0, 0.0),
		vec4(0.0, 0.0, 1.0, 0.0),
		vec4(0.0, 0.0, 0.0, 1.0)
	);
	mat4 transform = mat4 (
		vec4(1.0, 0.0, 0.0, 0.0),
		vec4(0.0, 1.0, 0.0,-0.95),
		vec4(0.0, 0.0, 1.0, 0.0),
		vec4(0.0, 0.0, 0.0, 1.0)
	);
    fragTexCoord = vertTexCoord;
    gl_Position = vec4(vert, 1) * transform * window_scale;
}
` + "\x00"

var TextFragmentShader = `
#version 330
uniform sampler2D tex;
in vec2 fragTexCoord;
out vec4 outputColor;
void main() {
	outputColor = texture(tex, fragTexCoord);
}
` + "\x00"

var squareVerts = []float32{
	-1.0, 1.0, 0.1, 0.0, 0.0,
	-1.0, -1.0, 0.1, 0.0, 1.0,
	1.0, 1.0, 0.1, 1.0, 0.0,
	1.0, -1.0, 0.1, 1.0, 1.0,
}

func main() {
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Samples, 6)
	window, err := glfw.CreateWindow(windowWidth, windowHeight, "fakeupdate", glfw.GetPrimaryMonitor(), nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	var text_vao uint32
	gl.GenVertexArrays(1, &text_vao)
	gl.BindVertexArray(text_vao)

	text_program, err := newProgram(TextVertexShader, TextFragmentShader)
	if err != nil {
		panic(err)
	}

	var text_vbo uint32
	gl.GenBuffers(1, &text_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, text_vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(squareVerts)*4, gl.Ptr(squareVerts), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(text_program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 5*4, 0)

	textureUniform := gl.GetUniformLocation(text_program, gl.Str("tex\x00"))
	gl.Uniform1i(textureUniform, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(text_program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, 5*4, 3*4)

	window_size := []float32{float32(windowHeight) / 100.0, float32(windowWidth) / 100.0}
	sizeUniform := gl.GetUniformLocation(text_program, gl.Str("window_size\x00"))
	gl.Uniform2fv(sizeUniform, 1, &window_size[0])

	gl.BindFragDataLocation(text_program, 0, gl.Str("outputColor\x00"))

	// Load the font
	fontBytes, err := f.ReadFile("noto.ttf")
	if err != nil {
		panic(err)
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		panic(err)
	}
	const imgW, imgH = 1920, 1920
	rgba := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	fg := image.White

	var size, dpi float64 = 8, 500
	d := &font.Drawer{
		Dst: rgba,
		Src: fg,
		Face: truetype.NewFace(f, &truetype.Options{
			Size:    size,
			DPI:     dpi,
			Hinting: font.HintingFull,
		}),
	}
	dy := int(math.Ceil(size * 1.5 * dpi / 72))
	titleTop := "Working on updates"
	y := 10 + int(math.Ceil(size*dpi/72))
	d.Dot = fixed.Point26_6{
		X: (fixed.I(imgW) - d.MeasureString(titleTop)) / 2,
		Y: fixed.I(y),
	}
	d.DrawString(titleTop)
	titleBottom := "0% complete."
	y += dy
	d.Dot = fixed.Point26_6{
		X: (fixed.I(imgW) - d.MeasureString(titleBottom)) / 2,
		Y: fixed.I(y),
	}
	d.DrawString(titleBottom)

	texture, err := newTextureRGBA(rgba)
	if err != nil {
		panic(err)
	}

	// Configure the vertex data
	var dot_vao uint32
	gl.GenVertexArrays(1, &dot_vao)
	gl.BindVertexArray(dot_vao)

	// Configure the vertex and fragment shaders
	dot_program, err := newProgram(DotVertexShader, DotFragmentShader)
	if err != nil {
		panic(err)
	}

	view := mgl32.LookAtV(mgl32.Vec3{3, 3, 3}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
	cameraUniform := gl.GetUniformLocation(dot_program, gl.Str("view\x00"))
	gl.UniformMatrix4fv(cameraUniform, 1, false, &view[0])

	gl.ClearColor(0, 0.352, 0.619, 1.0)
	gl.BindFragDataLocation(dot_program, 0, gl.Str("outputColor\x00"))

	circleVertices := []float32{}
	for i := 0.0; i < 361.0; i += 1.0 {
		r := float32(1.0)
		x := float32(r * float32(math.Sin(i)))
		y := float32(r * float32(math.Cos(i)))
		circleVertices = append(circleVertices, x)
		circleVertices = append(circleVertices, y)
		circleVertices = append(circleVertices, 0.0)
	}

	var dots_vbo uint32
	gl.GenBuffers(1, &dots_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, dots_vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(circleVertices)*4, gl.Ptr(circleVertices), gl.STATIC_DRAW)

	vertAttrib = uint32(gl.GetAttribLocation(dot_program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 0, 0)

	window_size = []float32{float32(windowHeight) / 100.0, float32(windowWidth) / 100.0}
	sizeUniform = gl.GetUniformLocation(dot_program, gl.Str("window_size\x00"))
	gl.Uniform2fv(sizeUniform, 1, &window_size[0])

	scale := mgl32.Scale2D(0.1, 0.1)
	scaleUniform := gl.GetUniformLocation(dot_program, gl.Str("scale\x00"))
	gl.UniformMatrix4fv(scaleUniform, 1, false, &scale[0])

	u_time := float32(0)
	timeUniform := gl.GetUniformLocation(dot_program, gl.Str("time\x00"))
	gl.Uniform1fv(timeUniform, 1, &u_time)

	gl.Enable(gl.MULTISAMPLE)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	for !window.ShouldClose() {
		windowWidth, windowHeight = window.GetSize()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.UseProgram(dot_program)
		// Render dots
		window_size := []float32{float32(windowHeight), float32(windowWidth)}
		sizeUniform := gl.GetUniformLocation(dot_program, gl.Str("window_size\x00"))
		gl.Uniform2fv(sizeUniform, 1, &window_size[0])

		scale := mgl32.Scale2D(0.009, 0.009)
		scaleUniform := gl.GetUniformLocation(dot_program, gl.Str("scale\x00"))
		gl.UniformMatrix3fv(scaleUniform, 1, false, &scale[0])

		u_time := float32(glfw.GetTime())
		timeUniform := gl.GetUniformLocation(dot_program, gl.Str("time\x00"))
		gl.Uniform1fv(timeUniform, 1, &u_time)

		gl.BindVertexArray(dot_vao)

		gl.DrawArraysInstanced(gl.TRIANGLE_FAN, 0, 361, 6)

		// Render text
		gl.UseProgram(text_program)

		sizeUniform = gl.GetUniformLocation(text_program, gl.Str("window_size\x00"))
		gl.Uniform2fv(sizeUniform, 1, &window_size[0])

		textureUniform = gl.GetUniformLocation(text_program, gl.Str("tex\x00"))
		gl.Uniform1i(textureUniform, 0)

		gl.BindVertexArray(text_vao)

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture)

		gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

		// Maintenance
		window.SwapBuffers()

		// Don't poll events ;)
		//glfw.PollEvents()
		gl.Viewport(0, 0, int32(windowWidth), int32(windowHeight))
	}
}
