// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

#import <Foundation/Foundation.h>
#import <LocalAuthentication/LocalAuthentication.h>

#include <stdlib.h>
#include <string.h>

static void set_error(char **error_message, NSError *error) {
    if (error_message == NULL || error == nil) {
        return;
    }

    const char *message = error.localizedDescription.UTF8String;
    if (message != NULL) {
        *error_message = strdup(message);
    }
}

int spore_authenticate(const char *reason, char **error_message) {
    @autoreleasepool {
        LAContext *context = [[LAContext alloc] init];
        NSError *error = nil;
        LAPolicy policy = LAPolicyDeviceOwnerAuthentication;

        if (![context canEvaluatePolicy:policy error:&error]) {
            set_error(error_message, error);
            return -1;
        }

        __block BOOL confirmed = NO;
        __block NSError *authentication_error = nil;
        dispatch_semaphore_t completed = dispatch_semaphore_create(0);
        NSString *localized_reason = [NSString stringWithUTF8String:reason];

        [context evaluatePolicy:policy
                localizedReason:localized_reason
                          reply:^(BOOL success, NSError *reply_error) {
            confirmed = success;
            authentication_error = reply_error;
            dispatch_semaphore_signal(completed);
        }];

        dispatch_semaphore_wait(completed, DISPATCH_TIME_FOREVER);
        if (confirmed) {
            return 1;
        }

        // A denied or cancelled prompt is an ordinary "no" response.
        (void)authentication_error;
        return 0;
    }
}
