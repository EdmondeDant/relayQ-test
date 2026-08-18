package service

var motion20FastVideoRoute = newLeonardoVideoRoute(leonardoVideoParameterSpec{
	model: "motion_2.0-fast", durationOptional: true, defaultWidth: 512, defaultHeight: 768, normalizeSize: func(_, _ int) (int, int) { return 512, 768 },
	startFrame: true,
})
