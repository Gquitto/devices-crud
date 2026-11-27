"""Insomnia collection for the Devices API.

The scenarios below are the single source for api/insomnia.json (Insomnia
export format v4). Edit them here and regenerate; do not edit the JSON by hand.

Usage (from the repository root; Python 3 standard library only):
    python3 scripts/insomnia.py [OUTPUT]          # default: api/insomnia.json
    python3 scripts/insomnia.py --verify [URL]    # default: http://localhost:8080

--verify runs every request in order against a running API and checks the
expected status codes. It creates devices and deletes them again in the
Teardown folder, so point it at a local or throwaway database.

Or use the Makefile: `make insomnia` and `make insomnia-verify`.
"""

import json
import re
import sys
import urllib.error
import urllib.request

UNKNOWN_ID = "00000000-0000-4000-8000-000000000000"

# ---- scripts (Insomnia >= 9 after-response scripts) ----


# Every string that goes into JavaScript passes through js(), which emits a
# double-quoted, escaped literal. Never interpolate text between quotes by
# hand: a quote inside a test name breaks the whole script.
def js(s):
    return json.dumps(s)


def status(code):
    return (f"insomnia.test({js(f'status is {code}')}, () => "
            f"insomnia.expect(insomnia.response.code).to.equal({code}));")


ERROR_BODY = ("insomnia.test('body has an error message', () => "
              "insomnia.expect(insomnia.response.json().error).to.be.a('string'));")


def save_id(var):
    return f"insomnia.environment.set({js(var)}, insomnia.response.json().id);"


def field(name, value):
    return (f"insomnia.test({js(f'{name} is {value}')}, () => "
            f"insomnia.expect(insomnia.response.json()[{js(name)}]).to.equal({js(value)}));")


def every(desc, expr):
    return (f"insomnia.test({js(desc)}, () => insomnia.expect("
            f"insomnia.response.json().every(d => {expr})).to.equal(true));")


def includes(var):
    return (f"insomnia.test({js(f'includes {var}')}, () => insomnia.expect("
            f"insomnia.response.json().some(d => d.id === insomnia.environment.get({js(var)}))).to.equal(true));")


EMPTY_ARRAY = ("insomnia.test('body is []', () => "
               "insomnia.expect(insomnia.response.json()).to.eql([]));")

LOCATION = ("insomnia.test('Location points to the new device', () => "
            "insomnia.expect(insomnia.response.headers.get('Location'))"
            ".to.equal('/devices/' + insomnia.response.json().id));")

KEEPS_CREATION_TIME = ("insomnia.test('creationTime is unchanged', () => "
                       "insomnia.expect(insomnia.response.json().creationTime)"
                       ".to.equal(insomnia.environment.get('available_created')));")

SAVE_CREATED = "insomnia.environment.set('available_created', insomnia.response.json().creationTime);"


def bad(*extra):
    return [status(400), ERROR_BODY, *extra]


# ---- scenarios: (folder, [(name, method, path, body, expected, scripts)]) ----

A, U, I = "{{ _.available_id }}", "{{ _.in_use_id }}", "{{ _.inactive_id }}"

FOLDERS = [
    ("00 Health", [
        ("Health check", "GET", "/health", None, 200, [status(200)]),
    ]),
    ("01 Setup - run first", [
        ("Create available (state omitted, defaults to available)", "POST", "/devices",
         {"name": "TestDevice", "brand": "TestBrand1"}, 201,
         [status(201), LOCATION, field("state", "available"), save_id("available_id"), SAVE_CREATED]),
        ("Create in-use", "POST", "/devices",
         {"name": "promax", "brand": "TestBrand2", "state": "in-use"}, 201,
         [status(201), field("state", "in-use"), save_id("in_use_id")]),
        ("Create inactive", "POST", "/devices",
         {"name": "RogAllyX", "brand": "TestBrand1", "state": "inactive"}, 201,
         [status(201), field("state", "inactive"), save_id("inactive_id")]),
    ]),
    ("02 Create - validation", [
        ("Blank name", "POST", "/devices", {"name": "", "brand": "BrandTest"}, 400, bad()),
        ("Whitespace-only brand", "POST", "/devices", {"name": "DevTest", "brand": " "}, 400, bad()),
        ("Missing brand", "POST", "/devices", {"name": "DevTest"}, 400, bad()),
        ("Invalid state", "POST", "/devices",
         {"name": "DevTest", "brand": "BrandTest", "state": "broken"}, 400, bad()),
        ("Unknown field id", "POST", "/devices", {"id": "x", "name": "A", "brand": "B"}, 400, bad()),
        ("Unknown field creationTime", "POST", "/devices",
         {"name": "A", "brand": "B", "creationTime": "2020-01-01T00:00:00Z"}, 400, bad()),
        ("Malformed JSON", "POST", "/devices", '{"name":', 400, bad()),
    ]),
    ("03 Fetch", [
        ("Get one", "GET", f"/devices/{A}", None, 200, [status(200), field("name", "TestDevice")]),
        ("Get unknown id", "GET", f"/devices/{UNKNOWN_ID}", None, 404, [status(404), ERROR_BODY]),
        ("Get invalid id", "GET", "/devices/not-a-uuid", None, 400, bad()),
        ("List all", "GET", "/devices", None, 200,
         [status(200), includes("available_id"), includes("in_use_id"), includes("inactive_id")]),
        ("Filter by brand (partial, case-insensitive)", "GET", "/devices?brand=brand1", None, 200,
         [status(200), includes("available_id"), includes("inactive_id"),
          every("every brand contains 'brand1'", "d.brand.toLowerCase().includes('brand1')")]),
        ("Filter by name (partial, case-insensitive)", "GET", "/devices?name=PROMAX", None, 200,
         [status(200), includes("in_use_id"),
          every("every name contains 'promax'", "d.name.toLowerCase().includes('promax')")]),
        ("Filter by state", "GET", "/devices?state=in-use", None, 200,
         [status(200), includes("in_use_id"), every("every state is in-use", "d.state === 'in-use'")]),
        ("Filters combined (AND)", "GET", "/devices?brand=brand1&state=inactive", None, 200,
         [status(200), includes("inactive_id"),
          every("every device is inactive with a brand containing 'brand1'",
                "d.brand.toLowerCase().includes('brand1') && d.state === 'inactive'")]),
        ("Filter with no match returns []", "GET", "/devices?brand=nokia", None, 200,
         [status(200), EMPTY_ARRAY]),
        ("Invalid state filter", "GET", "/devices?state=broken", None, 400, bad()),
    ]),
    ("04 Full update (PUT)", [
        ("Update all fields", "PUT", f"/devices/{I}",
         {"name": "DevTestUp", "brand": "BrandTestUp", "state": "available"}, 200,
         [status(200), field("name", "DevTestUp"), field("brand", "BrandTestUp"), field("state", "available")]),
        ("Omitted state defaults to available", "PUT", f"/devices/{I}",
         {"name": "DevTestUp", "brand": "BrandTestUp"}, 200, [status(200), field("state", "available")]),
        ("creationTime is kept", "PUT", f"/devices/{A}",
         {"name": "TestDevice", "brand": "TestBrand1", "state": "available"}, 200,
         [status(200), KEEPS_CREATION_TIME]),
        ("In-use: same name and brand is allowed", "PUT", f"/devices/{U}",
         {"name": "promax", "brand": "TestBrand2", "state": "in-use"}, 200, [status(200)]),
        ("In-use: rename is a conflict", "PUT", f"/devices/{U}",
         {"name": "Other", "brand": "TestBrand2", "state": "in-use"}, 409, [status(409), ERROR_BODY]),
        ("In-use: free and rename together is a conflict", "PUT", f"/devices/{U}",
         {"name": "Other", "brand": "TestBrand2", "state": "available"}, 409, [status(409), ERROR_BODY]),
        ("In-use: omitted state is rejected", "PUT", f"/devices/{U}",
         {"name": "promax", "brand": "TestBrand2"}, 400, bad()),
        ("Blank brand", "PUT", f"/devices/{A}", {"name": "TestDevice", "brand": " "}, 400, bad()),
        ("Invalid state", "PUT", f"/devices/{A}",
         {"name": "TestDevice", "brand": "TestBrand1", "state": "broken"}, 400, bad()),
        ("Unknown field creationTime", "PUT", f"/devices/{A}",
         {"name": "A", "brand": "B", "creationTime": "2020-01-01T00:00:00Z"}, 400, bad()),
        ("Unknown id", "PUT", f"/devices/{UNKNOWN_ID}",
         {"name": "A", "brand": "B", "state": "available"}, 404, [status(404), ERROR_BODY]),
        ("Invalid id", "PUT", "/devices/not-a-uuid", {"name": "A", "brand": "B"}, 400, bad()),
    ]),
    ("05 Partial update (PATCH)", [
        ("Only name (brand and state kept)", "PATCH", f"/devices/{A}", {"name": "DevTestUp"}, 200,
         [status(200), field("name", "DevTestUp"), field("brand", "TestBrand1"), field("state", "available"),
          KEEPS_CREATION_TIME]),
        ("Only state", "PATCH", f"/devices/{I}", {"state": "inactive"}, 200,
         [status(200), field("name", "DevTestUp"), field("state", "inactive")]),
        ("In-use: same name is allowed", "PATCH", f"/devices/{U}", {"name": "promax"}, 200, [status(200)]),
        ("In-use: rename is a conflict", "PATCH", f"/devices/{U}", {"name": "Other"}, 409,
         [status(409), ERROR_BODY]),
        ("Empty body", "PATCH", f"/devices/{A}", {}, 400, bad()),
        ("Blank name", "PATCH", f"/devices/{A}", {"name": ""}, 400, bad()),
        ("Invalid state", "PATCH", f"/devices/{A}", {"state": "broken"}, 400, bad()),
        ("Unknown field id", "PATCH", f"/devices/{A}", {"id": "x"}, 400, bad()),
        ("Unknown id", "PATCH", f"/devices/{UNKNOWN_ID}", {"name": "A"}, 404, [status(404), ERROR_BODY]),
        ("Invalid id", "PATCH", "/devices/not-a-uuid", {"name": "A"}, 400, bad()),
    ]),
    ("06 Delete", [
        ("In-use is a conflict", "DELETE", f"/devices/{U}", None, 409, [status(409), ERROR_BODY]),
        ("Delete available", "DELETE", f"/devices/{A}", None, 204, [status(204)]),
        ("Get after delete is 404", "GET", f"/devices/{A}", None, 404, [status(404), ERROR_BODY]),
        ("Delete again is 404", "DELETE", f"/devices/{A}", None, 404, [status(404), ERROR_BODY]),
        ("Invalid id", "DELETE", "/devices/not-a-uuid", None, 400, bad()),
    ]),
    ("07 Teardown - run last", [
        ("Free the in-use device", "PATCH", f"/devices/{U}", {"state": "available"}, 200,
         [status(200), field("state", "available")]),
        ("Delete the former in-use device", "DELETE", f"/devices/{U}", None, 204, [status(204)]),
        ("Delete the inactive device", "DELETE", f"/devices/{I}", None, 204, [status(204)]),
    ]),
]

# ---- Insomnia export ----


def build():
    wrk = "wrk_devices_api"
    resources = [
        {"_id": wrk, "_type": "workspace", "parentId": None, "name": "Devices API",
         "description": "Manual test collection for the Devices API. Run the folders in order "
                        "(01 Setup first, 07 Teardown last); Setup stores the device ids in the "
                        "environment. Tests run as after-response scripts (Insomnia 9+).",
         "scope": "collection"},
        {"_id": "env_devices_base", "_type": "environment", "parentId": wrk, "name": "Base Environment",
         "data": {"base_url": "http://localhost:8080", "available_id": "", "in_use_id": "",
                  "inactive_id": "", "available_created": ""}},
    ]
    n = 0
    for fi, (folder, requests) in enumerate(FOLDERS):
        fld = f"fld_{fi:02d}"
        resources.append({"_id": fld, "_type": "request_group", "parentId": wrk, "name": folder,
                          "metaSortKey": -1000 + fi})
        for ri, (name, method, path, body, expected, scripts) in enumerate(requests):
            n += 1
            req = {"_id": f"req_{fi:02d}_{ri:02d}", "_type": "request", "parentId": fld,
                   "name": name, "method": method, "url": "{{ _.base_url }}" + path,
                   "description": f"Expected: {expected}", "metaSortKey": -1000 + ri,
                   "headers": [], "body": {}, "parameters": [],
                   "afterResponseScript": "\n".join(scripts)}
            if body is not None:
                text = body if isinstance(body, str) else json.dumps(body, indent=2)
                req["body"] = {"mimeType": "application/json", "text": text}
                req["headers"] = [{"name": "Content-Type", "value": "application/json"}]
            resources.append(req)
    export = {"_type": "export", "__export_format": 4, "__export_source": "devices-crud",
              "resources": resources}
    return export, n


# ---- live verification ----


def verify(base):
    env = {}
    ok = fail = 0
    for folder, requests in FOLDERS:
        for name, method, path, body, expected, _ in requests:
            url = base + re.sub(r"\{\{ _\.(\w+) \}\}", lambda m: env.get(m.group(1), ""), path)
            data = None
            if body is not None:
                data = (body if isinstance(body, str) else json.dumps(body)).encode()
            req = urllib.request.Request(url, data=data, method=method,
                                         headers={"Content-Type": "application/json"} if data else {})
            try:
                with urllib.request.urlopen(req) as r:
                    code, raw = r.status, r.read()
            except urllib.error.HTTPError as e:
                code, raw = e.code, e.read()
            if method == "POST" and code == 201:
                var = {"Create available (state omitted, defaults to available)": "available_id",
                       "Create in-use": "in_use_id", "Create inactive": "inactive_id"}.get(name)
                if var:
                    env[var] = json.loads(raw)["id"]
            mark = "ok  " if code == expected else "FAIL"
            ok, fail = (ok + 1, fail) if code == expected else (ok, fail + 1)
            if code != expected:
                print(f"{mark} [{folder}] {name}: got {code}, want {expected} {raw[:200]!r}")
    print(f"{ok} passed, {fail} failed")
    return fail == 0


def main(args):
    if args and args[0] == "--verify":
        base = (args[1] if len(args) > 1 else "http://localhost:8080").rstrip("/")
        return 0 if verify(base) else 1
    out = args[0] if args else "api/insomnia.json"
    export, n = build()
    with open(out, "w") as f:
        json.dump(export, f, indent=2)
        f.write("\n")
    print(f"wrote {n} requests in {len(FOLDERS)} folders to {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
