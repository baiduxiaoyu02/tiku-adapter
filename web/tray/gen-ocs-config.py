import json

BS = chr(92)  # backslash
NL_ESC = BS + "n"  # backslash + n (2 chars) — the escape sequence in the handler source

config = [{
    "name": "TikuAdapter+视觉",
    "url": "http://localhost:8061/api/search",
    "homepage": "https://github.com/DokiDoki1103/tikuAdapter",
    "method": "post",
    "type": "GM_xmlhttpRequest",
    "contentType": "json",
    "headers": {},
    "data": {
        "question": "${title}",
        "options": {"handler": "return (env)=>env.options?.split('" + NL_ESC + "')"},
        "type": {"handler": "return (env)=> env.type === 'single' ? 0 : env.type === 'multiple' ? 1 : env.type === 'completion' ? 2 : env.type === 'judgement' ? 3 : 4"},
        "images": {"handler": "return (env) => env.images"},
        "suggestion_title": "${suggestion_title}",
        "suggestion_options": "${suggestion_options}"
    },
    "handler": "return (res)=>res.answer.allAnswer.map(i=>([res.question,i.join('#')]))"
}]

path = 'D:/user/Zcode/xubo/.zcode/workspace/default/tiku-adapter/web/ocs-config.json'
with open(path, 'w', encoding='utf-8', newline='\n') as f:
    json.dump(config, f, ensure_ascii=False, indent=2)

h = config[0]["data"]["options"]["handler"]
print("handler repr:", repr(h))
print("contains backslash+n:", NL_ESC in h)
print("contains real newline:", "\n" in h)
