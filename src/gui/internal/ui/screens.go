package ui

import (
	"errors"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/mauzec/tdsoftgui/internal/client"
)

const (
	ScreenLogin ScreenID = "login"

	ScreenTODO ScreenID = "todo"
)

// RegisterDefaultScreens registers the default screens.
func RegisterDefaultScreens(r *Router) {
	r.Register(ScreenLogin, loginScreen)
	r.Register(ScreenMain, mainScreen)

	r.Register(ScreenTODO, TODOScreen)

}

func loginScreen(r *Router) fyne.CanvasObject {
	var cl *client.Client
	_ = r.GetServiceAs(&cl)
	// ctx := r.ScreenContext()

	cl.StartCreatorServer()

	apiIDEntry, apiHashEntry := widget.NewEntry(), widget.NewEntry()

	phoneEntry, codeEntry := widget.NewEntry(), widget.NewEntry()
	phoneEntry.SetPlaceHolder("+12345678987")
	codeEntry.SetPlaceHolder("123456")
	phoneRow := container.NewBorder(nil, nil,
		widget.NewLabel("Phone:"), nil,
		phoneEntry,
	)
	codeRow := container.NewBorder(nil, nil,
		widget.NewLabel("Code:"), nil,
		codeEntry,
	)
	phoneEntry.Disable()
	phoneRow.Hide()
	codeEntry.Disable()
	codeRow.Hide()

	passwordEntry := widget.NewEntry()
	passwordEntry.SetPlaceHolder("password")
	passwordRow := container.NewBorder(nil, nil,
		widget.NewLabel("Password:"), nil,
		passwordEntry,
	)
	passwordEntry.Disable()
	passwordRow.Hide()

	authStep := 0

	backButton := widget.NewButton("Back", func() {})
	nextButton := widget.NewButton("Next", func() {})

	nextButton.OnTapped = func() {
		switch authStep {
		case 0:
			if apiIDEntry.Text == "" || apiHashEntry.Text == "" {
				return
			}
			cl.APIID = apiIDEntry.Text
			cl.APIHash = apiHashEntry.Text

			err := cl.SendAPIData()
			if err != nil {
				// TODO: add error message to UI
				log.Println("failed to send API data:", err)
				return
			}

			apiIDEntry.Disable()
			apiHashEntry.Disable()

			phoneEntry.Enable()
			phoneRow.Show()

			backButton.Enable()

			authStep = 1

		case 1:
			// TODO: add validation
			if phoneEntry.Text == "" {
				return
			}

			err := cl.SendPhone(phoneEntry.Text)
			if err != nil {
				log.Println("failed to send phone:", err)
				return
			}

			phoneEntry.Disable()

			codeEntry.Enable()
			codeRow.Show()
			authStep = 2

		case 2:
			if codeEntry.Text == "" {
				return
			}

			codeEntry.Disable()

			err := cl.SignIn(phoneEntry.Text, codeEntry.Text)
			if err != nil {
				if !errors.Is(err, client.ErrPasswordNeeded) {
					log.Println("failed to sign in:", err)
					return
				}

				passwordEntry.Enable()
				passwordRow.Show()
				authStep = 3
				return
			}

			if err := cl.SaveAPIConfig(); err != nil {
				log.Println("failed to save api config; deleting session...:", err)
				cl.DeleteSession()
				return
			}

			r.Show(ScreenMain)

		case 3:
			if passwordEntry.Text == "" {
				return
			}

			passwordEntry.Disable()

			err := cl.CheckPassword(passwordEntry.Text)
			if err != nil {
				log.Println("failed to check password:", err)
				return
			}

			if err := cl.SaveAPIConfig(); err != nil {
				log.Println("failed to save api config; deleting session...:", err)
				cl.DeleteSession()
				return
			}

			r.Show(ScreenMain)
		}
	}

	backButton.Disable()
	backButton.OnTapped = func() {
		switch authStep {
		case 1:
			apiIDEntry.Enable()
			apiHashEntry.Enable()
			backButton.Disable()

			phoneEntry.Disable()
			phoneRow.Hide()

		case 2:

		}
		authStep -= 1
	}

	return container.NewVBox(
		container.NewBorder(nil, nil,
			widget.NewLabel("API ID:"), nil,
			apiIDEntry,
		),
		container.NewBorder(nil, nil,
			widget.NewLabel("API Hash:"), nil,
			apiHashEntry,
		),
		phoneRow, codeRow, passwordRow,

		container.NewBorder(
			nil, nil, backButton, nextButton,
		),
	)
}

// TODOScreen is a placeholder screen for future implement.
//
//	Parameters: msg string
func TODOScreen(r *Router) fyne.CanvasObject {
	var msg string
	_ = r.ParamAs(ScreenTODO, &msg)

	return container.NewVBox(
		widget.NewLabel("TODO: " + msg),
	)
}
