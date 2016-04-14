import json
import sys
import re

def norm(s):
	s = s.replace(".", "_")
	return s

with open('./scheme_45.json') as o:
	j = json.loads(norm(o.read()))

types = {}
same = []

for c in j ['constructors']:
	if c['type'] in types:
		types[c['type']].append(c['predicate'])
	else:
		types[c['type']] = [c['predicate']]
	if c['type'].lower() == c['predicate'].lower():
		same.append(c['predicate'])

dele = []
for t in types:
	if len(types[t]) == 1 or not any([i.lower() == t.lower() for i in types[t]]):
		dele.append(t)

for d in dele:
	del types[d]

print(types)

f = open(sys.argv[1], "r+").read()
w = open(sys.argv[1], "w+")

for t in types:
	s = t[0].lower() + t[1:]
	f = re.sub("m\.(Vector_%s)\(\)" % s, "m.Vector() /* \\1 */", f)
	f = re.sub("m\.Object\(\)(\.\(TL_%s\))" % s, "m.Object() /* \\1 */", f)
	f = re.sub("(\w\s+)((\[\])?TL_%s)\n" % s, "\\1 \\3TL // \\2\n", f)
	f = re.sub("(Vector_%s)(\(e\.\w+\))" % s, "Vector\\2 // \\1", f)

w.write(f)
