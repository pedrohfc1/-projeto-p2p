package tray

func ShouldHide(closeToTray, available, quitting bool) bool {
	return closeToTray && available && !quitting
}
