package service

var kling30VideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "kling-3.0", durations: integerRange(3, 15), defaultWidth: 1920, defaultHeight: 1080, normalizeSize: preserveAllowedVideoSizes([][2]int{{1280, 720}, {720, 1280}, {960, 960}, {1920, 1080}, {1080, 1920}, {1440, 1440}, {3840, 2160}, {2160, 3840}, {2880, 2880}}, normalizeVideoSize(1280, 720, 960)),
	motionHasAudio: boolPointer(true), mode: "RESOLUTION_720", startFrame: true, endFrame: true,
})
