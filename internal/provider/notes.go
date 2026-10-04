package provider

// profilePartNotes returns the documentation notes shared by the resources
// that manage one part of a profile, such as its security settings or its
// denylist. typeName is the resource type, for example "nextdns_security".
// On the Terraform Registry, a paragraph starting with "~>" is shown as a
// highlighted note.
func profilePartNotes(typeName string) string {
	return "~> Use only one `" + typeName + "` resource per profile. Two resources for the same profile, " +
		"in the same or in different Terraform configurations, would keep overwriting each other.\n\n" +
		"~> To manage an existing profile, import this resource first and review `terraform plan` before applying, " +
		"to see what will change."
}
