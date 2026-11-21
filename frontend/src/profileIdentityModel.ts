export const fonts = [
 {id:'inter', name:'Inter'},
 {id:'poppins', name:'Poppins'},
 {id:'montserrat', name:'Montserrat'},
 {id:'space-grotesk', name:'Space Grotesk'},
 {id:'jetbrains-mono', name:'JetBrains Mono'},
] as const;
export type DisplayFontID = typeof fonts[number]['id'];
export const fontId = (id: string | null | undefined): DisplayFontID => fonts.find(font => font.id === id)?.id ?? 'inter';
export const profileNameLimit = 32;

const unsafeNameCharacters = /[\p{Cc}\u061c\u200e\u200f\u2028-\u202e\u2066-\u2069]/gu;
export function editName(value: string) {
 return Array.from(value.normalize('NFC').replace(unsafeNameCharacters, '')).slice(0,profileNameLimit).join('');
}
export const cleanName = (value: string) => editName(value).trim();

export function initials(name: string) {
 return name.trim().split(/\s+/u).slice(0,2).map(word => Array.from(word)[0] ?? '').join('').toLocaleUpperCase('pt-BR') || 'P';
}

export const imageInputLimit = 5 * 1024 * 1024;
export const imageAssetLimit = 256 * 1024;
export const imageOutputSide = 256;
export type ImageMime = 'image/png'|'image/jpeg'|'image/webp';
export type ImageAsset = {mime: ImageMime; data: string};

const supportedMimes = new Set<string>(['image/png','image/jpeg','image/webp']);
const hashPattern = /^[a-f0-9]{64}$/;
export const validAssetHash = (hash?:string) => typeof hash==='string' && hashPattern.test(hash);

export function imageAssetURL(asset: ImageAsset | null | undefined): string | undefined {
 if (!asset || !supportedMimes.has(asset.mime) || typeof asset.data !== 'string' || !asset.data.length ||
  asset.data.length > Math.ceil(imageAssetLimit/3)*4 || !/^[A-Za-z0-9+/]+={0,2}$/.test(asset.data)) return undefined;
 return 'data:' + asset.mime + ';base64,' + asset.data;
}

type ImageHeader = {mime:ImageMime; width:number; height:number};
const pngMagic = [137,80,78,71,13,10,26,10];
const startOfFrame = new Set([0xc0,0xc1,0xc2,0xc3,0xc5,0xc6,0xc7,0xc9,0xca,0xcb,0xcd,0xce,0xcf]);

function jpegDimensions(bytes:Uint8Array, view:DataView):ImageHeader|undefined {
 for(let offset=2;offset+3<bytes.length;) {
  if(bytes[offset++]!==0xff) return;
  while(bytes[offset]===0xff) offset++;
  const marker=bytes[offset++];
  if(marker===0xda || marker===0xd9 || offset+2>bytes.length) return;
  const length=view.getUint16(offset);
  if(length<2 || offset+length>bytes.length) return;
  if(startOfFrame.has(marker)) {
   if (length>=8) return {mime:'image/jpeg',width:view.getUint16(offset+5),height:view.getUint16(offset+3)};
   return;
  }
  offset+=length;
 }
}

function webpDimensions(bytes:Uint8Array,view:DataView):ImageHeader|undefined {
 const chunk = String.fromCharCode(...bytes.subarray(12,16));
 const chunkSize = view.getUint32(16,true);
 if (view.getUint32(4,true)+8!==bytes.length || chunkSize>bytes.length-20) return;
 if(chunk==='VP8X'&&chunkSize===10) {
  const width=view.getUint16(24,true)+(bytes[26]<<16)+1;
  const height=view.getUint16(27,true)+(bytes[29]<<16)+1;
  return {mime:'image/webp',width,height};
 }
 if(chunk==='VP8L'&&chunkSize>=5&&bytes[20]===0x2f) {
  const bits=view.getUint32(21,true);
  return {mime:'image/webp',width:(bits&0x3fff)+1,height:((bits>>>14)&0x3fff)+1};
 }
 if(chunk==='VP8 '&&chunkSize>=10&&bytes[23]===0x9d&&bytes[24]===1&&bytes[25]===0x2a) {
  return {mime:'image/webp',width:view.getUint16(26,true)&0x3fff,height:view.getUint16(28,true)&0x3fff};
 }
}

// Limit bytes and decoded pixels before invoking the browser decoder.
export function inspectImage(buffer:ArrayBuffer):ImageHeader {
 if(buffer.byteLength>imageInputLimit) throw Error('A imagem excede o limite de 5 MB.');
 const bytes=new Uint8Array(buffer);
 const view=new DataView(buffer);
 let header:ImageHeader|undefined;
 if(bytes.length>=24&&pngMagic.every((n,i)=>bytes[i]===n)&&view.getUint32(8)===13&&view.getUint32(12)===0x49484452) {
  header={mime:'image/png',width:view.getUint32(16),height:view.getUint32(20)};
 } else if(bytes.length>=4&&bytes[0]===0xff&&bytes[1]===0xd8) {
  header=jpegDimensions(bytes,view);
 } else if(bytes.length>=25&&view.getUint32(0)===0x52494646&&view.getUint32(8)===0x57454250) {
  header=webpDimensions(bytes,view);
 }
 if(!header) throw Error('Arquivo inválido. Escolha uma imagem PNG, JPEG ou WebP.');
 if(!header.width||!header.height||header.width>8192||header.height>8192||header.width*header.height>32_000_000) {
  throw Error('Dimensões da imagem acima do limite. Escolha uma imagem menor.');
 }
 return header;
}

export type Crop = {zoom:number; x:number; y:number};
export const centerCrop:Crop = {zoom:1,x:0,y:0};
export function squareCrop(width:number,height:number,crop:Crop) {
 const size=Math.min(width,height)/Math.min(4,Math.max(1,crop.zoom));
 const position=(value:number)=> (Math.max(-100,Math.min(100,value))+100)/200;
 return {x:(width-size)*position(crop.x),y:(height-size)*position(crop.y),size};
}
