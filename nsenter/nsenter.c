/* Copyright 2018 HyperHQ Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>

/* Get all of the CLONE_NEW* flags. */
#include "namespace.h"

int nsenter(int nsPid)
{
	int nsFd, ret;
	char *path;

	path = malloc(sizeof(char)*64);
	if (path == NULL)
		return -1;
	sprintf(path, "/proc/%d/ns/net", nsPid);
	nsFd = open(path, O_RDONLY);
	if (nsFd < 0) {
		free(path);
		return -1;
	}
	ret = setns(nsFd, CLONE_NEWNET);

	free(path);
	close(nsFd);
	return ret;
}
