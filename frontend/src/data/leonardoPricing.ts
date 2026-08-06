export interface PricingConfig {
  quality?: string | null
  size?: string | null
  duration?: number | null
  resolution?: string | null
  cost?: number
  min?: number
  max?: number
  minCost?: number
  maxCost?: number
}

export interface PricingModel {
  qualities?: Array<string | null>
  sizes?: Array<string | null>
  ratios?: string[]
  durations?: Array<number | null>
  resolutions?: Array<string | null>
  slider?: boolean
  configs: PricingConfig[]
}

type PricingCatalog = Record<'image' | 'video', Record<string, PricingModel>>

export const PRICING: PricingCatalog = { image: {}, video: {} }
const A = ['2:3', '1:1', '16:9']
const Z = [null]
const img = (name: string, qualities: Array<string | null>, sizes: Array<string | null>, ratios: string[], costs: number[]) => {
  PRICING.image[name] = { qualities, sizes, ratios, configs: qualities.flatMap((quality, qualityIndex) => sizes.map((size, sizeIndex) => ({ quality, size, cost: costs[qualityIndex * sizes.length + sizeIndex] }))) }
}
img('GPT Image 2',['Low','Medium','High'],['Small 1376×768','Medium 2048×1136','Large 3584×2016'],A,[.012,.0254,.0762,.0987,.2153,.6683,.3902,.8566,2.6596])
img('Nano Banana 2',Z,['Small 1376×768','Medium 2752×1536','Large 5504×3072'],A,[.0389,.0583,.0777])
img('Nano Banana 2 Lite',Z,Z,[],[.0449])
img('Krea 2 Turbo',Z,['Small 1376×768','Medium 2064×1152','Large 2752×1536'],A,[.012,.0284,.0508])
img('Seedream 5.0 Pro',Z,Z,[],[.0568])
img('Lucid Origin',['Fast','Ultra'],['Small 1376×768','Medium 1600×896','Large 1920×1088'],A,[.012,.0164,.0254,.0628,.0852,.1256])
img('Nano Banana Pro',Z,['Small 1376×768','Medium 2752×1536','Large 5504×3072'],A,[.2093,.2093,.3738])
;['Ideogram 4.0','Ideogram 3.0'].forEach(name => img(name,['Low','Medium','High'],Z,[],[.0374,.0748,.1121]))
img('Recraft V4',Z,Z,[],[.0598]); img('Recraft V4 Pro',Z,Z,[],[.3738]); img('Seedream 4.5',Z,Z,[],[.0419])
img('FLUX.2 Pro',Z,Z,['2:3','1:1','16:9','9:16'],[.0478])
img('GPT Image-1.5',['Low','Medium','High'],Z,['2:3','1:1','3:2'],[.0194,.0568,.2048])
img('Seedream 4.0',Z,Z,[],[.0419]); img('Nano Banana',Z,Z,[],[.0389])
img('Lucid Realism',['Fast','Ultra'],['Small 1024×1024','Medium 1200×1200','Large 1440×1440'],A,[.012,.0179,.0254,.0628,.0867,.1241])
img('FLUX.1 Kontext Max',Z,Z,[],[.1495]); img('FLUX.1 Kontext',Z,Z,[],[.0748])
img('FLUX Dev',Z,['Small 896×896','Medium 1024×1024','Large 1120×1120'],A,[.009,.012,.015])
img('FLUX Schnell',Z,['Small 896×896','Medium 1024×1024','Large 1120×1120'],A,[.003,.003,.0045])
;['Phoenix 1.0','Phoenix 0.9'].forEach(name => img(name,['Fast','Quality','Ultra'],['Small 896×896','Medium 1024×1024','Large 1120×1120'],A,[.015,.015,.015,.0224,.0224,.0224,.0538,.0658,.0748]))
img('Anime',['Fast','Quality'],['Small 888×888','Medium 960×960','Large 1024×1024'],A,[.003,.0045,.0045,.015,.015,.015])
img('Cinematic Kino',['Fast','Quality'],['Small 896×896','Medium 1024×1024','Large 1120×1120'],A,[.0045,.006,.006,.0239,.0239,.0269])
;['Concept Art','Graphic Design','Illustrative Albedo','Leonardo Lightning','Lifelike Vision','Portrait Perfect','Stock Photography'].forEach(name => img(name,['Fast','Quality'],['Small 888×888','Medium 960×960','Large 1024×1024'],A,[.0045,.0045,.006,.0239,.0239,.0239]))
const vid = (name: string, durations: Array<number | null>, resolutions: Array<string | null>, ratios: string[], costs: number[]) => {
  PRICING.video[name] = { durations, resolutions, ratios, configs: durations.flatMap((duration, durationIndex) => resolutions.map((resolution, resolutionIndex) => ({ duration, resolution, cost: costs[durationIndex * resolutions.length + resolutionIndex] }))) }
}
const slider = (name: string, min: number, max: number, resolutions: Array<string | null>, ratios: string[], endpoints: number[][]) => {
  PRICING.video[name] = { durations: Array.from({ length: max - min + 1 }, (_, index) => min + index), resolutions, ratios, slider: true, configs: resolutions.map((resolution, index) => ({ resolution, min, max, minCost: endpoints[index][0], maxCost: endpoints[index][1] })) }
}
slider('MiniMax H3',5,15,Z,[],[[.897,2.691]])
vid('Hailuo 2.3',[6,10],Z,[],[.2691,.5606])
slider('Gemini Omni Flash',3,10,Z,[],[[.3588,1.196]])
slider('Seedance 2.0 Mini',4,15,['Standard 864×496','HD 1280×720'],['16:9','1:1','9:16'],[[.3588,1.3455],[.7774,2.9153]])
slider('Grok Imagine 1.5',3,15,['Standard 736×400','HD 1280×720','Full HD 1888×1072'],['1:1','16:9','9:16'],[[.4037,2.0183],[.7176,3.588],[1.0091,5.0456]])
slider('Happy Horse 1.1',3,15,Z,[],[[.8522,4.2608]]); slider('Wan 2.7',2,10,Z,[],[[.1645,.8223]]); slider('Kling 3.0 Turbo',3,15,Z,[],[[.7176,3.588]])
vid('Wan 2.6',[5,10,15],Z,[],[.314,.6279,.9419])
slider('Seedance 2.0',4,15,['Standard 864×496','HD 1280×720','Full HD 1920×1080','4K 3840×2160'],['16:9','1:1','9:16'],[[.8402,3.153],[1.8075,6.7813],[4.0679,15.258],[11.3859,42.6972]])
slider('Seedance 2.0 Fast',4,15,['Standard 864×496','HD 1280×720'],['16:9','1:1','9:16'],[[.6713,2.5221],[1.4457,5.4239]])
slider('Happy Horse 1.0',3,15,Z,[],[[.6279,3.1395]]); slider('Kling Video 3.0',3,15,Z,[],[[.5651,2.8256]]); slider('Kling Video O3 Omni',3,15,Z,[],[[1.0046,5.0232]])
vid('Veo 3.1 Fast',[4,6,8],['HD 720p','Full HD 1080p','4K 2160p'],[],[.897,.897,2.093,1.3455,1.3455,3.1395,1.794,1.794,4.186])
vid('Veo 3.1 Lite',[4,6,8],Z,[],[.299,.4485,.598]); vid('Kling 2.6',[5,10],Z,[],[.903,1.806]); vid('Kling O1 Video Model',[5,10],Z,[],[.755,1.51]); vid('Hailuo 2.3 Fast',[6,10],Z,[],[.1914,.3214])
vid('Veo 3.1',[4,6,8],['HD 720p','Full HD 1080p','4K 2160p'],[],[2.392,2.392,3.588,3.588,3.588,5.382,4.784,4.784,7.176])
vid('Kling 2.5 Turbo',[5,10],Z,[],[.3513,.7027]); vid('Kling 2.5 Turbo Standard',[5,10],Z,[],[.2841,.5681])
vid('Seedance 1.0 Pro Fast',[4,6,8,10],['Standard 864×480','HD 1248×704','Full HD 1920×1088'],['4:3','16:9','1:1','9:16'],[.0449,.1047,.2691,.0748,.1645,.3887,.1047,.2243,.5233,.1346,.2841,.6578])
vid('Seedance 1.0 Pro',[4,6,8,10],['Standard 864×480','HD 1248×704','Full HD 1920×1088'],['4:3','16:9','1:1','9:16'],[.1346,.2841,.6578,.1944,.4186,.9867,.2542,.5532,1.3156,.3289,.6877,1.6445])
vid('Kling 2.1 Pro',[5,10],Z,[],[.613,1.2259]); vid('LTX-2.3 Pro',[6,8,10],Z,[],[.5382,.7176,.897]); vid('LTX-2 Pro',[6,8],Z,[],[.4844,.6458])
slider('LTX-2.3 Fast',6,20,Z,[],[[.3588,1.196]]); vid('LTX-2 Fast',[6,8],Z,[],[.3229,.4306]); vid('Motion 2.0',[null],Z,[],[.299]); vid('Motion 2.0 Fast',[null],Z,[],[.1047])
