/*
 * SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

#include "defs.h"

int func3() { return (1 << 6) | func1(); }
