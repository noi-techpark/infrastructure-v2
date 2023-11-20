// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package com.opendatahub;

import org.apache.camel.builder.RouteBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import jakarta.enterprise.context.ApplicationScoped;

/**
 * Global error handler.
 */
@ApplicationScoped
public class ErrorHandler extends RouteBuilder {

    private Logger LOG = LoggerFactory.getLogger(ErrorHandler.class);

    @Override
    public void configure() throws Exception {
        onCompletion()
                .onFailureOnly()
                .process(exchange -> LOG.error("{}", exchange.getMessage().getBody()));
    }
}
