package service

var seedance20FastVideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "seedance-2.0-fast", durations: integerRange(4, 15), defaultWidth: 1280, defaultHeight: 720, normalizeSize: preserveAllowedVideoSizes([][2]int{{864, 496}, {496, 864}, {640, 640}, {1280, 720}, {720, 1280}, {960, 960}}, normalizeVideoSize(864, 496, 640)),
	seed: true, startFrame: true, endFrame: true, maxReferences: 4, referenceStrength: true,
})
