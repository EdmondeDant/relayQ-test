package service

var wan27VideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "wan-2.7", durations: integerRange(2, 10), defaultWidth: 1280, defaultHeight: 720, normalizeSize: preserveAllowedVideoSizes([][2]int{{1280, 720}, {720, 1280}, {960, 960}, {1920, 1080}, {1080, 1920}, {1440, 1440}}, normalizeVideoSize(1280, 720, 960)),
	startFrame: true, endFrame: true, maxReferences: 6, referenceStrength: true, resolution: wanVideoResolution,
})
