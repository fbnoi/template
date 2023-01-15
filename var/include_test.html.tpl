{% set show_content3 = 1 %}
{% set content3 = "content 3" %}
{% if show_content3 %}
    <p>{{ content3 }}</p>
{% else %}
    <p>not show content3</p>
{% endif %}