#!/usr/bin/python
#
# Run from src/emailservice: python -m unittest email_server_test

import unittest
from unittest import mock

import grpc

import demo_pb2
import email_server


def make_order():
  return demo_pb2.OrderResult(
    order_id='o-1',
    shipping_tracking_id='AB-12345-6789012345',
    shipping_cost=demo_pb2.Money(currency_code='USD', units=8, nanos=990000000),
    shipping_address=demo_pb2.Address(
      street_address='1600 Amphitheatre Pkwy', city='Mountain View',
      state='CA', country='US', zip_code=94043),
    items=[demo_pb2.OrderItem(
      item=demo_pb2.CartItem(product_id='OLJCESPC7Z', quantity=2),
      cost=demo_pb2.Money(currency_code='USD', units=19, nanos=990000000))])


class ConfirmationTemplateTest(unittest.TestCase):
  def test_renders_tracking_id_and_address(self):
    body = email_server.template.render(order=make_order())
    self.assertIn('Tracking #AB-12345-6789012345', body)
    self.assertIn('1600 Amphitheatre Pkwy, Mountain View, US 94043', body)

  def test_missing_field_fails_send_with_internal(self):
    broken = email_server.env.from_string('Tracking #{{ order.tracking_id }}')
    request = demo_pb2.SendOrderConfirmationRequest(email='a@b.c', order=make_order())
    context = mock.Mock()
    with mock.patch.object(email_server, 'template', broken), \
         mock.patch.object(email_server.EmailService, 'send_email') as send_email:
      email_server.EmailService.SendOrderConfirmation(None, request, context)
    context.set_code.assert_called_once_with(grpc.StatusCode.INTERNAL)
    send_email.assert_not_called()


if __name__ == '__main__':
  unittest.main()
