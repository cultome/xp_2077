package ui

import (
	"fmt"
)

func (m AppModel) viewEnvCheck() string {
	title := m.headerLine("3NV_CH3CK :: sys_req")
	lines := []string{title, "", m.styles.Subtle.Render("sc4nn1ng r3qu1r3d v4r1abl3s..."), ""}

	if len(m.envReport.Statuses) == 0 {
		lines = append(lines, m.styles.Subtle.Render("pr3ss r t0 sc4n 4g41n."))
	} else {
		for _, st := range m.envReport.Statuses {
			icon := m.styles.Success.Render("[0K]")
			if !st.Present {
				icon = m.styles.Error.Render("[M1SS1NG]")
			}
			lines = append(lines, fmt.Sprintf("%s %s", icon, st.Name))
			if !st.Present {
				lines = append(lines, m.styles.Subtle.Render("  -> "+st.Hint))
			}
		}
	}

	lines = append(lines, "")
	if m.envReport.Missing {
		lines = append(lines, m.styles.Error.Render("r3q v4r1abl3s n0t f0und."))
		lines = append(lines, m.styles.Subtle.Render("f1x 3nv + pr3ss R t0 r3try."))
	} else if len(m.envReport.Statuses) > 0 {
		lines = append(lines, m.styles.Success.Render("4ll r3qu1r3m3nts s4t1sf13d."))
		lines = append(lines, m.styles.Subtle.Render("j4ck1ng 1nt0 l04d p1p3l1n3..."))
	}

	lines = append(lines, "", m.styles.Footer.Render("R: r3try   ENTER: c0nt1nu3   Q: qu1t"))
	return m.screen(lines)
}
