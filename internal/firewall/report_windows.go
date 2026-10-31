package firewall

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func number(d *ole.IDispatch, name string) int64 {
	v, e := oleutil.GetProperty(d, name)
	if e != nil {
		return 0
	}
	defer v.Clear()
	return v.Val
}
func text(d *ole.IDispatch, name string) string {
	v, e := oleutil.GetProperty(d, name)
	if e != nil {
		return ""
	}
	defer v.Clear()
	return v.ToString()
}
func Inspect(port int, exe string) Report {
	r := Report{Port: port, Summary: "Leitura do firewall indisponível. Não foi feita nenhuma alteração; confira as regras manualmente se o peer não responder."}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); e != nil {
		return r
	}
	defer ole.CoUninitialize()
	o, e := oleutil.CreateObject("HNetCfg.FwPolicy2")
	if e != nil {
		return r
	}
	defer o.Release()
	policy, e := o.QueryInterface(ole.IID_IDispatch)
	if e != nil {
		return r
	}
	defer policy.Release()
	profiles := number(policy, "CurrentProfileTypes")
	rules, e := oleutil.GetProperty(policy, "Rules")
	if e != nil {
		return r
	}
	defer rules.Clear()
	count := 0
	e = oleutil.ForEach(rules.ToIDispatch(), func(v *ole.VARIANT) error {
		count++
		if count > 4096 {
			return errors.New("limite de regras")
		}
		rule := v.ToIDispatch()
		if rule == nil || number(rule, "Enabled") == 0 || number(rule, "Direction") != 1 || number(rule, "Profiles")&profiles == 0 {
			return nil
		}
		protocol := number(rule, "Protocol")
		if protocol != 6 && protocol != 256 {
			return nil
		}
		app := text(rule, "ApplicationName")
		if app != "" && !strings.EqualFold(app, exe) {
			return nil
		}
		if protocol == 6 && !portMatch(text(rule, "LocalPorts"), port) {
			return nil
		}
		switch number(rule, "Action") {
		case 1:
			r.PotentialAllowRules++
		case 0:
			r.PotentialBlockRules++
		}
		return nil
	})
	if e != nil {
		return r
	}
	r.Available = true
	r.Summary = fmt.Sprintf("Leitura de regras de entrada TCP %d: %d permissões e %d bloqueios potencialmente compatíveis. Perfis ativos: %d. Restrições de IP/interface e políticas adicionais podem mudar o resultado; só o teste remoto confirma o caminho. Nenhuma regra alterada.", port, r.PotentialAllowRules, r.PotentialBlockRules, profiles)
	return r
}
