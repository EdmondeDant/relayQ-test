package service

var kling25TurboStandardVideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "kling-2.5-turbo-standard", durations: []int{5, 10}, defaultWidth: 1280, defaultHeight: 720, normalizeSize: preserveAllowedVideoSizes([][2]int{{1280, 720}, {720, 1280}, {960, 960}}, normalizeVideoSize(1280, 720, 960)),
	startFrame: true, startFrameRequired: true,
})
