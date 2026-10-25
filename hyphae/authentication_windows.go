// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	winapi "golang.org/x/sys/windows"
)

const (
	roInitMultithreaded = 1

	consentVerified = iota
	consentDeviceNotPresent
	consentNotConfiguredForUser
	consentDisabledByPolicy
	consentDeviceBusy
	consentRetriesExhausted
	consentCanceled

	asyncStarted = iota
	asyncCompleted
	asyncCanceled
	asyncError

	authenticationTimeout = 90 * time.Second
)

var (
	combaseDLL                     = winapi.NewLazySystemDLL("combase.dll")
	roInitialize                   = combaseDLL.NewProc("RoInitialize")
	roUninitialize                 = combaseDLL.NewProc("RoUninitialize")
	roGetActivationFactory         = combaseDLL.NewProc("RoGetActivationFactory")
	windowsCreateString            = combaseDLL.NewProc("WindowsCreateString")
	windowsDeleteString            = combaseDLL.NewProc("WindowsDeleteString")
	user32DLL                      = winapi.NewLazySystemDLL("user32.dll")
	createWindowEx                 = user32DLL.NewProc("CreateWindowExW")
	destroyWindow                  = user32DLL.NewProc("DestroyWindow")
	kernel32DLL                    = winapi.NewLazySystemDLL("kernel32.dll")
	getModuleHandle                = kernel32DLL.NewProc("GetModuleHandleW")
	userConsentVerifierRuntimeName = "Windows.Security.Credentials.UI.UserConsentVerifier"
	userConsentVerifierInteropIID  = winapi.GUID{Data1: 0x39e050c3, Data2: 0x4e74, Data3: 0x441a, Data4: [8]byte{0x8d, 0xc0, 0xb8, 0x11, 0x04, 0xdf, 0x94, 0x9c}}
	asyncConsentResultIID          = winapi.GUID{Data1: 0xfd596ffd, Data2: 0x2318, Data3: 0x558f, Data4: [8]byte{0x9d, 0xbe, 0xd2, 0x1d, 0xf4, 0x37, 0x64, 0xa5}}
	asyncInfoIID                   = winapi.GUID{Data1: 0x00000036, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

func authenticateUser(reason string) (bool, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := initializeWindowsRuntime(); err != nil {
		return false, err
	}
	defer roUninitialize.Call()

	owner, err := createAuthenticationWindow()
	if err != nil {
		return false, err
	}
	defer destroyWindow.Call(owner)

	message, err := newHString(reason)
	if err != nil {
		return false, err
	}
	defer windowsDeleteString.Call(message)

	factory, err := activationFactory()
	if err != nil {
		return false, err
	}
	defer comRelease(factory)

	operation, err := requestVerification(factory, owner, message)
	if err != nil {
		return false, err
	}
	defer comRelease(operation)

	result, err := waitForConsentResult(operation)
	if err != nil {
		return false, err
	}
	return interpretConsentResult(result)
}

func initializeWindowsRuntime() error {
	result, _, callErr := roInitialize.Call(roInitMultithreaded)
	if failedHRESULT(result) {
		return hresultError("RoInitialize", result, callErr)
	}
	return nil
}

func createAuthenticationWindow() (uintptr, error) {
	className, err := winapi.UTF16PtrFromString("STATIC")
	if err != nil {
		return 0, err
	}
	windowName, err := winapi.UTF16PtrFromString("Spore Hyphae Authentication")
	if err != nil {
		return 0, err
	}
	instance, _, callErr := getModuleHandle.Call(0)
	if instance == 0 {
		return 0, fmt.Errorf("GetModuleHandleW: %w", callErr)
	}

	const wsOverlappedWindow = 0x00cf0000
	window, _, callErr := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		wsOverlappedWindow,
		0,
		0,
		0,
		0,
		0,
		0,
		instance,
		0)
	if window == 0 {
		return 0, fmt.Errorf("CreateWindowExW: %w", callErr)
	}
	return window, nil
}

func newHString(value string) (uintptr, error) {
	utf16, err := winapi.UTF16FromString(value)
	if err != nil {
		return 0, err
	}
	var hstring uintptr
	result, _, callErr := windowsCreateString.Call(
		uintptr(unsafe.Pointer(&utf16[0])),
		uintptr(len(utf16)-1),
		uintptr(unsafe.Pointer(&hstring)))
	if failedHRESULT(result) {
		return 0, hresultError("WindowsCreateString", result, callErr)
	}
	return hstring, nil
}

func activationFactory() (unsafe.Pointer, error) {
	runtimeName, err := newHString(userConsentVerifierRuntimeName)
	if err != nil {
		return nil, err
	}
	defer windowsDeleteString.Call(runtimeName)

	var factory unsafe.Pointer
	result, _, callErr := roGetActivationFactory.Call(
		runtimeName,
		uintptr(unsafe.Pointer(&userConsentVerifierInteropIID)),
		uintptr(unsafe.Pointer(&factory)))
	if failedHRESULT(result) {
		return nil, hresultError("RoGetActivationFactory", result, callErr)
	}
	return factory, nil
}

func requestVerification(factory unsafe.Pointer, owner uintptr, message uintptr) (unsafe.Pointer, error) {
	var operation unsafe.Pointer
	result := comCall(
		factory,
		6,
		owner,
		message,
		uintptr(unsafe.Pointer(&asyncConsentResultIID)),
		uintptr(unsafe.Pointer(&operation)))
	if failedHRESULT(result) {
		return nil, hresultError("RequestVerificationForWindowAsync", result, nil)
	}
	return operation, nil
}

func waitForConsentResult(operation unsafe.Pointer) (int32, error) {
	asyncInfo, err := comQueryInterface(operation, &asyncInfoIID)
	if err != nil {
		return 0, err
	}
	defer comRelease(asyncInfo)

	deadline := time.Now().Add(authenticationTimeout)
	for {
		var status int32
		result := comCall(asyncInfo, 7, uintptr(unsafe.Pointer(&status)))
		if failedHRESULT(result) {
			return 0, hresultError("IAsyncInfo.get_Status", result, nil)
		}

		switch status {
		case asyncCompleted:
			var consentResult int32
			result = comCall(operation, 8, uintptr(unsafe.Pointer(&consentResult)))
			if failedHRESULT(result) {
				return 0, hresultError("IAsyncOperation.GetResults", result, nil)
			}
			_ = comCall(asyncInfo, 10)
			return consentResult, nil
		case asyncCanceled:
			return consentCanceled, nil
		case asyncError:
			var operationError uintptr
			result = comCall(asyncInfo, 8, uintptr(unsafe.Pointer(&operationError)))
			if failedHRESULT(result) {
				return 0, hresultError("IAsyncInfo.get_ErrorCode", result, nil)
			}
			return 0, fmt.Errorf("Windows authentication failed with HRESULT 0x%08x", uint32(operationError))
		case asyncStarted:
			if time.Now().After(deadline) {
				_ = comCall(asyncInfo, 9)
				return 0, fmt.Errorf("Windows authentication timed out")
			}
			time.Sleep(50 * time.Millisecond)
		default:
			return 0, fmt.Errorf("Windows authentication returned unknown async status %d", status)
		}
	}
}

func interpretConsentResult(result int32) (bool, error) {
	switch result {
	case consentVerified:
		return true, nil
	case consentCanceled:
		return false, nil
	case consentDeviceNotPresent:
		return false, fmt.Errorf("Windows Hello verification device is unavailable")
	case consentNotConfiguredForUser:
		return false, fmt.Errorf("Windows Hello is not configured for the current user")
	case consentDisabledByPolicy:
		return false, fmt.Errorf("Windows Hello verification is disabled by policy")
	case consentDeviceBusy:
		return false, fmt.Errorf("Windows Hello verification device is busy")
	case consentRetriesExhausted:
		return false, fmt.Errorf("Windows Hello verification retries were exhausted")
	default:
		return false, fmt.Errorf("Windows Hello returned unknown verification result %d", result)
	}
}

func comQueryInterface(object unsafe.Pointer, iid *winapi.GUID) (unsafe.Pointer, error) {
	var resultObject unsafe.Pointer
	result := comCall(
		object,
		0,
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&resultObject)))
	if failedHRESULT(result) {
		return nil, hresultError("IUnknown.QueryInterface", result, nil)
	}
	return resultObject, nil
}

func comRelease(object unsafe.Pointer) {
	if object != nil {
		comCall(object, 2)
	}
}

func comCall(object unsafe.Pointer, method int, arguments ...uintptr) uintptr {
	vtable := *(**[11]uintptr)(object)
	result, _, _ := syscall.SyscallN(vtable[method], append([]uintptr{uintptr(object)}, arguments...)...)
	return result
}

func failedHRESULT(result uintptr) bool {
	return int32(result) < 0
}

func hresultError(operation string, result uintptr, callErr error) error {
	if callErr != nil && callErr != syscall.Errno(0) {
		return fmt.Errorf("%s failed with HRESULT 0x%08x: %w", operation, uint32(result), callErr)
	}
	return fmt.Errorf("%s failed with HRESULT 0x%08x", operation, uint32(result))
}
