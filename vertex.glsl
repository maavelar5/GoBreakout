#version 300 es

layout (location = 0) in vec2 aVec;

layout (location = 1) in vec4 aModel1;
layout (location = 2) in vec4 aModel2;
layout (location = 3) in vec4 aModel3;
layout (location = 4) in vec4 aModel4;

layout (location = 5) in vec4 aColor;

out vec2 texCoords;
out vec4 uColor;

// uniform mat4 uModel;
uniform mat4 uProjection;

void main ()
{
    uColor    = aColor;
    texCoords = aVec;

    gl_Position = uProjection * mat4 (aModel1, aModel2, aModel3, aModel4)
                  * vec4 (aVec, 0.0, 1.0f);
}
