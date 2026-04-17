package planner

import "testing"

func TestClassify_DirectCoords(t *testing.T) {
	p := New()
	d := p.Classify("click at (450, 320)")
	if d.Engine != EngineEML {
		t.Errorf("Expected EML, got %s", d.Engine)
	}
}

func TestClassify_Click(t *testing.T) {
	p := New()
	d := p.Classify("haz clic en el boton de guardar")
	if d.Engine != EngineEML {
		t.Errorf("Expected EML for simple click, got %s", d.Engine)
	}
}

func TestClassify_Navigate(t *testing.T) {
	p := New()
	d := p.Classify("ve a http://google.com")
	if d.Engine != EngineEML {
		t.Errorf("Expected EML for navigation, got %s", d.Engine)
	}
}

func TestClassify_Complex_LLM(t *testing.T) {
	p := New()
	d := p.Classify("encuentra el boton de enviar y haz clic en el si el formulario esta completo")
	if d.Engine != EngineLLM {
		t.Errorf("Expected LLM for complex mission, got %s", d.Engine)
	}
}

func TestClassify_ShortKeyword(t *testing.T) {
	p := New()
	d := p.Classify("guardar archivo")
	if d.Engine != EngineEML {
		t.Errorf("Expected EML for short keyword, got %s", d.Engine)
	}
}

func TestStats(t *testing.T) {
	p := New()
	p.Classify("click at (100, 200)")   // EML
	p.Classify("haz clic en aceptar")   // EML
	p.Classify("find the submit button and verify the form is valid before clicking") // LLM

	if p.Stats.TotalMissions != 3 {
		t.Errorf("Expected 3 total, got %d", p.Stats.TotalMissions)
	}
	if p.Stats.EMLRouted != 2 {
		t.Errorf("Expected 2 EML, got %d", p.Stats.EMLRouted)
	}
	if p.Stats.LLMRouted != 1 {
		t.Errorf("Expected 1 LLM, got %d", p.Stats.LLMRouted)
	}
}

func TestEngineType_String(t *testing.T) {
	if EngineEML.String() != "EML" {
		t.Error("EngineEML string wrong")
	}
	if EngineLLM.String() != "LLM" {
		t.Error("EngineLLM string wrong")
	}
}
