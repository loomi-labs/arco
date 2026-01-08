#!/bin/bash

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 input_video output_gif"
  exit 1
fi

ffmpeg -i "$1" -vf "fps=10,scale=800:-1:flags=lanczos,palettegen" /tmp/palette.png
ffmpeg -i "$1" -i /tmp/palette.png -filter_complex "fps=10,scale=800:-1:flags=lanczos[x];[x][1:v]paletteuse" "$2"