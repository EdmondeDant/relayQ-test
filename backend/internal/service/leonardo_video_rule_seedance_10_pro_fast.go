package service

var seedance10ProFastVideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "seedance-1.0-pro-fast", durations: []int{4, 6, 8, 10}, defaultWidth: 1248, defaultHeight: 704, normalizeSize: preserveAllowedVideoSizes([][2]int{{864, 480}, {480, 864}, {640, 640}, {1248, 704}, {704, 1248}, {960, 960}, {1920, 1088}, {1088, 1920}, {1440, 1440}}, normalizeVideoSize(864, 480, 640)),
	seed: true, startFrame: true,
})
