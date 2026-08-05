package taitan

const (
	ChannelName         = "taitan-video"
	DefaultModel        = "seedance-2.0"
	SubmitEndpoint      = "/openapi/v1/video/generations"
	DefaultDuration     = 5
	DefaultAspectRatio  = "9:16"
	MaxReferenceCount   = 12
	MaxImageReferences  = 9
	MaxMediaReferences  = 3
	MaxImageReferenceMB = 10
	MaxVideoReferenceMB = 100
	MaxAudioReferenceMB = 50
)

var ModelList = []string{
	DefaultModel,
}

var supportedDurations = map[int]bool{
	5:  true,
	8:  true,
	10: true,
	12: true,
	15: true,
}

var supportedAspectRatios = map[string]bool{
	"9:16": true,
	"16:9": true,
}
