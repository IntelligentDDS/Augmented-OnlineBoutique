// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
const SERVICE_NAME = process.env.SERVICE_NAME;
const POD_NAME = process.env.POD_NAME;
const NODE_NAME = process.env.NODE_NAME;

const api = require('@opentelemetry/api');
const tracer = require('./tracer')((SERVICE_NAME));

const cardValidator = require('simple-card-validator');
const uuid = require('uuid/v4');
const pino = require('pino');

const logger = pino({
  name: 'paymentservice-charge',
  messageKey: 'message',
  changeLevelName: 'severity',
  timestamp: false, //pino.stdTimeFunctions.isoTime,
  useLevelLabels: true
});


class CreditCardError extends Error {
  constructor(message) {
    super(message);
    this.code = 400; // Invalid argument error
  }
}

class InvalidCreditCard extends CreditCardError {
  constructor(cardType) {
    super(`Credit card info is invalid`);
  }
}

class UnacceptedCreditCard extends CreditCardError {
  constructor(cardType) {
    super(`Sorry, we cannot process ${cardType} credit cards. Only VISA or MasterCard is accepted.`);
  }
}

class ExpiredCreditCard extends CreditCardError {
  constructor(number, month, year) {
    super(`Your credit card (ending ${number.substr(-4)}) expired on ${month}/${year}`);
  }
}

/**
 * Verifies the credit card number and (pretend) charges the card.
 *
 * @param {*} request
 * @return transaction_id - a random uuid v4.
 */
module.exports = function charge(request) {
  const currentSpan = api.trace.getSpan(api.context.active());

  // const currentSpan = tracer.startSpan('hipstershop.PaymentService/Charge', {
  //   parent: parentSpan,
  //   attributes: {
  //     "PodName": POD_NAME,
  //     "NodeName": NODE_NAME,
  //   },
  // });
  const spanId = currentSpan.spanContext().spanId;
  const traceId = currentSpan.spanContext().traceId;
  currentSpan.setAttribute("PodName", POD_NAME);
  currentSpan.setAttribute("NodeName", NODE_NAME);

  const { amount, credit_card: creditCard } = request;
  const cardNumber = creditCard.credit_card_number;
  const cardInfo = cardValidator(cardNumber);
  const {
    card_type: cardType,
    valid
  } = cardInfo.getCardDetails();

  if (!valid) { throw new InvalidCreditCard(); }

  // Only VISA and mastercard is accepted, other card types (AMEX, dinersclub) will
  // throw UnacceptedCreditCard error.
  if (!(cardType === 'visa' || cardType === 'mastercard')) { throw new UnacceptedCreditCard(cardType); }

  // Also validate expiration is > today.
  const currentMonth = new Date().getMonth() + 1;
  const currentYear = new Date().getFullYear();
  const { credit_card_expiration_year: year, credit_card_expiration_month: month } = creditCard;
  if ((currentYear * 12 + currentMonth) > (year * 12 + month)) { throw new ExpiredCreditCard(cardNumber.replace('-', ''), month, year); }

  logger.info(`TraceID: ${traceId} SpanID: ${spanId} Transaction processed: ${cardType} ending ${cardNumber.substr(-4)} \
    Amount: ${amount.currency_code}${amount.units}.${amount.nanos}`);

  return { transaction_id: uuid() };
};
