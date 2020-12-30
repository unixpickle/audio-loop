# audio-loop

This is a small program that finds the "perfect" timestamp at which to loop a song. It can be used to make a single song work as background music for a longer video.

# How it works

The program automatically tries every timestamp for looping, and chooses the one with the best overlap correlation. Imagine taking the audio clip and repeating it twice, and then moving the second clip over the first. When you do this, there is overlap between the two clips. If the waveform at the start of the second clip correlates strongly with the waveform it is replacing, then the loop will sound seamless. 

# Usage

In its simplest form, you can loop an audio file like this:

```
$ go run . -input input.mp3 -output output.mp3
```

There are additional flags which can be used to specify how the audio is looped. To see them, use the `-help` flag.
