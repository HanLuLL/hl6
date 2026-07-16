package auth

import "testing"

func TestNormalizeEmailTrimsAndNormalizesDomain(t *testing.T) {
	got, err := NormalizeEmail("  User.Name@EXAMPLE.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "user.name@example.com" {
		t.Fatalf("got %q, want %q", got, "user.name@example.com")
	}
}

func TestValidateRegistrationDomainHonorsExactAllowlist(t *testing.T) {
	policy := DomainPolicy{
		Mode:    DomainPolicyAllowlist,
		Domains: []string{"example.com"},
	}

	if err := ValidateRegistrationDomain("user@example.com", policy); err != nil {
		t.Fatalf("allowed domain rejected: %v", err)
	}
	if err := ValidateRegistrationDomain("user@outside.example", policy); err == nil {
		t.Fatal("unlisted domain accepted")
	}
	if err := ValidateRegistrationDomain("user@sub.example.com", policy); err == nil {
		t.Fatal("subdomain incorrectly matched exact allowlist entry")
	}
}

func TestValidateRegistrationDomainHonorsExactBlocklist(t *testing.T) {
	policy := DomainPolicy{
		Mode:    DomainPolicyBlocklist,
		Domains: []string{"blocked.example"},
	}

	if err := ValidateRegistrationDomain("user@example.com", policy); err != nil {
		t.Fatalf("unblocked domain rejected: %v", err)
	}
	if err := ValidateRegistrationDomain("user@blocked.example", policy); err == nil {
		t.Fatal("blocked domain accepted")
	}
	if err := ValidateRegistrationDomain("user@sub.blocked.example", policy); err != nil {
		t.Fatalf("subdomain incorrectly matched exact blocklist entry: %v", err)
	}
}

func TestValidateRegistrationDomainRejectsInvalidPolicyInputs(t *testing.T) {
	for _, test := range []struct {
		name   string
		email  string
		policy DomainPolicy
	}{
		{
			name:   "unknown mode",
			email:  "user@example.com",
			policy: DomainPolicy{Mode: "wildcard"},
		},
		{
			name:   "wildcard domain",
			email:  "user@example.com",
			policy: DomainPolicy{Mode: DomainPolicyAllowlist, Domains: []string{"*.example.com"}},
		},
		{
			name:   "malformed domain",
			email:  "user@example.com",
			policy: DomainPolicy{Mode: DomainPolicyAllowlist, Domains: []string{"bad_domain"}},
		},
		{
			name:   "malformed email",
			email:  "not an email",
			policy: DomainPolicy{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateRegistrationDomain(test.email, test.policy); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestValidateRegistrationDomainRejectsMalformedEmailDomainSpelling(t *testing.T) {
	for _, email := range []string{"123456@ qq.com", "123456@qq.com."} {
		if err := ValidateRegistrationDomain(email, DomainPolicy{}); err == nil {
			t.Fatalf("registration accepted malformed email domain spelling %q", email)
		}
	}
}

func TestQQAvatarURLUsesHTTPSForNumericQQEmail(t *testing.T) {
	got := QQAvatarURL(" 123456@qq.com ")
	want := "https://q.qlogo.cn/headimg_dl?dst_uin=123456&spec=640&img_type=jpg"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	for _, email := range []string{"name@qq.com", "1234@qq.com", "1234567890123@qq.com", "123456@qq.cn", "123456@foxmail.com", "12345@qq.com.evil"} {
		if got := QQAvatarURL(email); got != "" {
			t.Fatalf("unexpected QQ avatar for %q: %q", email, got)
		}
	}
}

func TestQQAvatarURLRejectsMalformedQQDomainSpelling(t *testing.T) {
	for _, email := range []string{"123456@ qq.com", "123456@qq.com."} {
		if got := QQAvatarURL(email); got != "" {
			t.Fatalf("got QQ avatar for malformed email domain spelling %q: %q", email, got)
		}
	}
}
