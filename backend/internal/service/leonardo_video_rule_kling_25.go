package service

var kling25VideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "kling-2.5", durations: []int{5, 10}, defaultWidth: 1920, defaultHeight: 1080, normalizeSize: preserveAllowedVideoSizes([][2]int{{1280, 720}, {720, 1280}, {960, 960}, {1920, 1080}, {1080, 1920}, {1440, 1440}}, normalizeVideoSize(1280, 720, 960)),
	startFrame: true, endFrame: true,
})
