package feature

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/KnicKnic/go-powershell/pkg/powershell"
)

type WindowsOptionalFeature struct {
	Name   string
	Status bool
}

func powershellScriptExecutor(powershell *powershell.Runspace, script string) powershell.Object {
	executeCommand := powershell.ExecScript(script, false, nil, "OS")
	defer executeCommand.Close()

	return executeCommand.Objects[0]
}

func isAdmin(powershell *powershell.Runspace) (bool, error) {
	adminCheckCommand := `
		$window = [System.Security.Principal.WindowsIdentity]::GetCurrent()
		$windowsPrincipal = new-object System.Security.Principal.WindowsPrincipal($window)
		$isAdmin = "true"
 
		$requestAdminRole = [System.Security.Principal.WindowsBuiltInRole]::Administrator
		if (-Not ($windowsPrincipal.IsInRole($requestAdminRole))) {
			$isAdmin = "false"
		}
		$($isAdmin)
	`

	executeAdminCheckCommand := powershellScriptExecutor(powershell, adminCheckCommand)

	result, err := strconv.ParseBool(executeAdminCheckCommand.ToString())

	if err != nil {
		return false, errors.New("unable to check for Administrator privileges")
	}

	return result, nil
}

// GetOptionalFeatureStatus returns the availability of Windows Optional Feature
func GetOptionalFeatureStatus(featureName string) (bool, error) {
	powerShellRunspace := powershell.CreateRunspaceSimple()
	defer powerShellRunspace.Close()

	isAdmin, err := isAdmin(&powerShellRunspace)

	if err != nil {
		return false, err
	}

	if !isAdmin {
		return false, errors.New("checking for feature requires Administrator privileges")
	}

	buildCommand := fmt.Sprintf(`
		$w_o_feature = Get-WindowsOptionalFeature -FeatureName %s -Online
		$($w_o_feature.State)
	`, featureName)

	executeCommand := powershellScriptExecutor(&powerShellRunspace, buildCommand)

	if executeCommand.IsNull() {
		return false, errors.New(fmt.Sprintf("unable to get status for %s", featureName))
	}

	if executeCommand.ToString() == "Disabled" {
		return false, nil
	}

	return true, nil
}

// GetMultipleOptionalFeaturesStatus returns a list of Windows Optional Features and their status
func GetMultipleOptionalFeaturesStatus(featuresNames []string) ([]WindowsOptionalFeature, error) {
	var featuresStatusContainer []WindowsOptionalFeature

	for _, featureName := range featuresNames {
		getFeatureStatus, err := GetOptionalFeatureStatus(featureName)
		if err != nil {
			return []WindowsOptionalFeature{}, err
		}

		featuresStatusContainer = append(featuresStatusContainer, WindowsOptionalFeature{
			Name:   featureName,
			Status: getFeatureStatus,
		})
	}

	return featuresStatusContainer, nil
}

// SetOptionalFeatureStatus sets the status of a Windows Optional Feature
func SetOptionalFeatureStatus(feature WindowsOptionalFeature, restart bool) (bool, error) {
	powerShellRunspace := powershell.CreateRunspaceSimple()
	defer powerShellRunspace.Close()

	isAdmin, err := isAdmin(&powerShellRunspace)

	if err != nil {
		return false, err
	}

	if !isAdmin {
		return false, errors.New("setting optional feature requires Administrator privileges")
	}

	var restartCommand string

	if restart {
		restartCommand = "Restart-Computer -Force"
	}

	buildCommand := fmt.Sprintf(`
		Disable-WindowsOptionalFeature -Online -FeatureName %s -NoRestart
		%s
	`, feature.Name, restartCommand)

	if feature.Status {
		buildCommand = fmt.Sprintf(`
			Enable-WindowsOptionalFeature -Online -FeatureName %s  -NoRestart -All
			%s
		`, feature.Name, restartCommand)
	}

	powershellScriptExecutor(&powerShellRunspace, buildCommand)

	return true, nil
}
