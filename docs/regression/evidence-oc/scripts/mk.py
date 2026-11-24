import json,sys,os
k=open(os.path.expanduser('~/.config/edupilot-team/secrets/opencode-go.key')).read().strip()
name=sys.argv[1]; key=k if sys.argv[2]=='real' else 'oc_sk_'+'Zx9Qw3'*8
base=sys.argv[3] if len(sys.argv)>3 else 'https://opencode.ai/inference/go/v1'
print(json.dumps({"type":"openai_compatible","name":name,"base_url":base,"api_key":key,"models":[{"model":"deepseek-v4.1-flash","kind":"chat","price_in":"0","price_out":"0"}]}))
