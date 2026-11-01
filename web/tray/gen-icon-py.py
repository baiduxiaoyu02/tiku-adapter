from PIL import Image, ImageDraw, ImageFont

SIZE = 512
RADIUS = 116
C_TOP = (99, 102, 241)     # indigo-500
C_BOTTOM = (139, 92, 246)  # violet-500
FONT = 'C:/Windows/Fonts/msyhbd.ttc'

# 1. 对角渐变底
grad = Image.new('RGBA', (SIZE, SIZE))
gd = ImageDraw.Draw(grad)
for y in range(SIZE):
    for x in range(SIZE):
        t = (x + y) / (2.0 * (SIZE - 1))
        r = int(C_TOP[0] + (C_BOTTOM[0] - C_TOP[0]) * t)
        g = int(C_TOP[1] + (C_BOTTOM[1] - C_TOP[1]) * t)
        b = int(C_TOP[2] + (C_BOTTOM[2] - C_TOP[2]) * t)
        gd.point((x, y), fill=(r, g, b, 255))

# 2. 圆角矩形蒙版
mask = Image.new('L', (SIZE, SIZE), 0)
md = ImageDraw.Draw(mask)
md.rounded_rectangle([0, 0, SIZE - 1, SIZE - 1], radius=RADIUS, fill=255)
grad.putalpha(mask)

# 3. 白色「题」字（微软雅黑粗体，光学居中）
glyph = Image.new('RGBA', (SIZE, SIZE), (0, 0, 0, 0))
gd2 = ImageDraw.Draw(glyph)
font = ImageFont.truetype(FONT, 296)
gd2.text((SIZE / 2, SIZE / 2 + 6), '题', font=font, fill=(255, 255, 255, 255), anchor='mm')
img = Image.alpha_composite(grad, glyph)

# 4. 输出：多尺寸 ICO + PNG 预览
ico_path = 'D:/user/Zcode/xubo/.zcode/workspace/default/tiku-adapter/web/tray/tiku.ico'
png_path = 'D:/user/Zcode/xubo/.zcode/workspace/default/tiku-adapter/web/tray/tiku-icon-preview.png'
img.save(ico_path, format='ICO', sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
img.save(png_path, format='PNG')
print('ICO:', ico_path)
print('PNG:', png_path)
