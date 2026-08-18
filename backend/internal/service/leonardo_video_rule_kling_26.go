package service

var kling26VideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "kling-2.6", durations: []int{5, 10}, defaultWidth: 1920, defaultHeight: 1080, normalizeSize: preserveAllowedVideoSizes([][2]int{{1920, 1080}, {1080, 1920}, {1440, 1440}}, normalizeVideoSize(1920, 1080, 1440)),
	motionHasAudio: boolPointer(true), startFrame: true,
})
