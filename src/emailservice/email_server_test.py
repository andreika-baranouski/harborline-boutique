#!/usr/bin/python
#
# Run from src/emailservice: python -m unittest email_server_test

import unittest
from unittest import mock

import grpc

import demo_pb2
import email_server


def make_order(estimated_delivery=''):
  return demo_pb2.OrderResult(
    order_id='o-1',
    shipping_tracking_id='AB-12345-6789012345',
    shipping_cost=demo_pb2.Money(currency_code='USD', units=8, nanos=990000000),
    shipping_address=demo_pb2.Address(
      street_address='1600 Amphitheatre Pkwy', city='Mountain View',
      state='CA', country='US', zip_code=94043),
    items=[demo_pb2.OrderItem(
      item=demo_pb2.CartItem(product_id='OLJCESPC7Z', quantity=2),
      cost=demo_pb2.Money(currency_code='USD', units=19, nanos=990000000))],
    estimated_delivery=estimated_delivery)


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


class EstimatedDeliveryTest(unittest.TestCase):
  def test_renders_estimate_after_tracking_id(self):
    body = email_server.template.render(order=make_order('2026-11-12'))
    self.assertIn('Estimated delivery: Thursday, November 12, 2026', body)
    self.assertLess(body.index('Tracking #AB-12345-6789012345'), body.index('Estimated delivery:'))

  def test_omits_line_without_estimate(self):
    body = email_server.template.render(order=make_order())
    self.assertIn('Tracking #AB-12345-6789012345', body)
    self.assertNotIn('Estimated delivery', body)

  def test_invalid_estimate_still_sends_without_line(self):
    request = demo_pb2.SendOrderConfirmationRequest(email='a@b.c', order=make_order('not-a-date'))
    context = mock.Mock()
    with mock.patch.object(email_server.EmailService, 'send_email') as send_email:
      email_server.EmailService.SendOrderConfirmation(mock.Mock(), request, context)
    context.set_code.assert_not_called()
    send_email.assert_called_once()
    self.assertNotIn('Estimated delivery', send_email.call_args.args[2])

  def test_format_delivery_date(self):
    self.assertEqual(email_server.format_delivery_date('2026-11-12'), 'Thursday, November 12, 2026')
    self.assertEqual(email_server.format_delivery_date('2027-01-04'), 'Monday, January 4, 2027')
    self.assertEqual(email_server.format_delivery_date(''), '')
    self.assertEqual(email_server.format_delivery_date('12/11/2026'), '')

  def test_other_locale_falls_back_to_en_us(self):
    self.assertEqual(email_server.format_delivery_date('2026-11-12', locale='fr-FR'), 'Thursday, November 12, 2026')


if __name__ == '__main__':
  unittest.main()
