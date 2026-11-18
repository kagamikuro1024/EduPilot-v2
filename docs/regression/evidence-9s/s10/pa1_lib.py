from pa import *
def review(qid, dec, C=C1, user="teacher"):
    v = api(user, "GET", f"/courses/{C}/questions/{qid}")[1]["version"]
    return api(user, "PUT", f"/courses/{C}/questions/{qid}/review", {"decision": dec, "version": v})
def approve(qid, C=C1, user="teacher"):
    c, j, _ = review(qid, "REQUEST", C, user); c2, j2, _ = review(qid, "APPROVE", C, user)
    return c, c2, j2
