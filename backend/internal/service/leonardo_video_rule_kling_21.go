package service

var kling21VideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "kling-2.1", durations: []int{5, 10}, defaultWidth: 1920, defaultHeight: 1080, normalizeSize: preserveAllowedVideoSizes([][2]int{{1920, 1080}, {1080, 1920}}, normalizeVideoSize(1920, 1080, 1440)),
	startFrame: true, endFrame: true, startFrameRequired: true,
})
