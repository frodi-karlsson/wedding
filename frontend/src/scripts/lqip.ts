import sharp from 'sharp';

export async function generateLqip(buffer: Buffer): Promise<string> {
  const lqip = await sharp(buffer)
    .resize(16, 16, { fit: 'cover' })
    .webp({ quality: 30 })
    .toBuffer();
  return lqip.toString('base64');
}
