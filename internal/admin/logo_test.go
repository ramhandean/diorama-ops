package admin

import (
	"strings"
	"testing"
)

func TestSanitizeSVG(t *testing.T) {
	maliciousSVG := `
		<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" onload="alert('xss')">
			<circle cx="50" cy="50" r="40" stroke="green" stroke-width="4" fill="yellow" />
			<script>alert('pwned');</script>
			<script type="text/javascript">document.location='http://evil.com'</script>
			<foreignObject width="100" height="100">
				<body xmlns="http://www.w3.org/1999/xhtml">
					<iframe src="http://evil.com"></iframe>
				</body>
			</foreignObject>
			<a href="javascript:alert(1)">Click me</a>
			<image xlink:href="http://evil.com/leak.png" />
		</svg>
	`

	clean, err := SanitizeSVG([]byte(maliciousSVG))
	if err != nil {
		t.Fatalf("sanitize failed: %v", err)
	}

	result := string(clean)

	if strings.Contains(result, "<script") {
		t.Errorf("svg still contains script tag")
	}
	if strings.Contains(result, "alert") {
		t.Errorf("svg still contains alert code")
	}
	if strings.Contains(result, "onload") {
		t.Errorf("svg still contains onload handler")
	}
	if strings.Contains(result, "foreignObject") {
		t.Errorf("svg still contains foreignObject")
	}
	if strings.Contains(result, "evil.com") {
		t.Errorf("svg still contains external link")
	}

	// Verify basic SVG shape remains
	if !strings.Contains(result, "<circle") {
		t.Errorf("expected clean circle shape to remain")
	}
}
