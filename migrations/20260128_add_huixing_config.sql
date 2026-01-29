INSERT INTO ai_service_configs (
    service_type,
    provider,
    name,
    base_url,
    endpoint,
    query_endpoint,
    api_key,
    model,
    is_active,
    priority,
    is_default,
    created_at,
    updated_at
) VALUES (
    'image',
    'huixing',
    '汇星云文生图',
    'http://azj1.dc.huixingyun.com:55875/webhook/72874db5-0e24-44cb-8f2c-7fa60435e652',
    'http://azj1.dc.huixingyun.com:55875/webhook/fba6d1a8-57af-405f-8752-8f88313d7c10', -- 使用endpoint字段存储查询接口
    '',
    'any', -- 不需要API Key
    '["8188"]', -- ComfyUI 端口
    1,
    10,
    0,
    '2026-01-28 10:48:00',
    '2026-01-28 10:48:00'
);
