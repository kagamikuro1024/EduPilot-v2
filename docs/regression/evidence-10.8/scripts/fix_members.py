from asl import *
S=st();C=S["cid"]
for n in range(7,13):
    u=f"sv{n:02d}"; c,j,_=api(u,"POST","/courses/join",{"code":"QCKASTV"}); print(u,c)
