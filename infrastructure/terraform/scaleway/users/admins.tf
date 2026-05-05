# This file creates a group and users for the noi digital team.

resource "scaleway_iam_user" "s_seppi" {
  username = "s.seppi"
  email    = "s.seppi@noi.bz.it"
}

resource "scaleway_iam_user" "c_zagler" {
  username = "c.zagler"
  email    = "c.zagler@noi.bz.it"
}

resource "scaleway_iam_user" "m_roggia" {
  username = "m.roggia"
  email    = "m.roggia@noi.bz.it"
}

data "scaleway_iam_group" "administrators" {
  name = "Administrators"
}

resource "scaleway_iam_group_membership" "s_seppi" {
  group_id = data.scaleway_iam_group.administrators.group_id
  user_id  = scaleway_iam_user.s_seppi.id
}

resource "scaleway_iam_group_membership" "c_zagler" {
  group_id = data.scaleway_iam_group.administrators.group_id
  user_id  = scaleway_iam_user.c_zagler.id
}

resource "scaleway_iam_group_membership" "m_roggia" {
  group_id = data.scaleway_iam_group.administrators.group_id
  user_id  = scaleway_iam_user.m_roggia.id
}
