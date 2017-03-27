SELECT id, name, plan_id, stripe_customer_id, stripe_subscription_id
FROM accounts
WHERE stripe_customer_id IS NULL OR stripe_customer_id = ''
ORDER BY name ASC;
