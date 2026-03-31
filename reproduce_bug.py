def test_concat():
    url = "http://test.com"
    text = "URL: " + url
    return text

def test_dict():
    d = {"success": True, "url": "http://test.com"}
    return d

class Response:
    def __init__(self, status, text):
        self.status_code = status
        self.text = text

def test_class():
    resp = Response(200, "OK")
    return resp.status_code

print("Concat:", test_concat())
print("Dict:", test_dict())
print("Class:", test_class())
