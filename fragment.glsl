#version 300 es

precision highp float;

in vec4 uColor;
in vec2 texCoords;

out vec4 color;

void main ()
{
    color = uColor;
}
